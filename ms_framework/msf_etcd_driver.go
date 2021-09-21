package ms_framework

import (
    // etcd "go.etcd.io/etcd/client/v3"
    etcdctl "go.etcd.io/etcd/clientv3"  // need grpc@v1.26.0
    "context"
    "time"
    "fmt"
    "strings"
    "strconv"
    "sync"
)

type EtcdDriver struct {
    mode     int8  // 0 - service regist  1 - service discover
    cli     *etcdctl.Client
    lease   *etcdctl.LeaseGrantResponse
    exit    bool
    leaseDV int
    mutex   sync.Mutex
}

func (e *EtcdDriver) Start() {
    cli, err := etcdctl.New(etcdctl.Config{
        Endpoints:   GlobalCfg.Etcd,
        // Endpoints:   []string {"127.0.0.1:2389"}, // for connect test
        DialTimeout: 5 * time.Second,
    })

    if err != nil {
        ERROR_LOG("etcd clientv3 new error %v %v", err, GlobalCfg.Etcd)
        return
    }

    // 到这一步并没有建立网络连接
    // DEBUG_LOG("etcd driver client: %+v", cli)

    e.cli = cli

    if 0 == e.mode {
        e.ServiceRegist()
        e.LeaseWatch()
    } else if 1 == e.mode {
        e.ServiceDiscover()
        e.ServiceWatch()
    }

    e.exit = false
}

func (e *EtcdDriver) Stop() {
    INFO_LOG("etcd driver stop...")

    if e.lease != nil {
        ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
        defer cancel()
        _, err := e.cli.Revoke(ctx, e.lease.ID)
        if err != nil {
            ERROR_LOG("etcd lease revoke error %v", err)
        }
    }

    if e.cli != nil {
        e.cli.Close()
    }

    e.exit = true
}

func (e *EtcdDriver) LeaseWatch() {

    timeCh := time.After(time.Second * 1)
    
    go func() {
        for !e.exit {
            select {
            case <- timeCh:
                ctx, cancelGet := context.WithTimeout(context.Background(), 5 * time.Second)
                rsp, err := e.cli.Get(ctx, e.GenEtcdServiceKey())
                if err != nil {
                    ERROR_LOG(fmt.Sprintf("lease watch %s error %v", e.GenEtcdServiceKey(), err))
                    cancelGet()
                    
                    timeCh = time.After(time.Second * 1)
                    continue
                }

                // INFO_LOG("etcd get key %s result %v-%s", e.GenEtcdServiceKey(), len(rsp.Kvs), rsp.Kvs)
                
                if len(rsp.Kvs) == 0 {
                    INFO_LOG("etcd key %s lease invaild, retry regist", e.GenEtcdServiceKey())
                    e.ServiceRegist()

                    timeCh = time.After(time.Second * 1)
                    continue
                }
                
                timeCh = time.After(time.Second * 5)
                cancelGet()
            }
        }
    } ()
}

func (e *EtcdDriver) ServiceRegist() {
    e.mutex.Lock()
    defer e.mutex.Unlock()

    // minimum lease ttl is 2 second
    // lease, err := e.cli.Grant(context.TODO(), 2)
    ctx, cancelGrant := context.WithTimeout(context.Background(), 5 * time.Second)
    defer cancelGrant()
    lease, err := e.cli.Grant(ctx, 2)
    if err != nil {
        ERROR_LOG("etcd grant error %v", err)
        return
    }

    e.lease = lease

    ctx, cancelPut := context.WithTimeout(context.Background(), 5 * time.Second)
    defer cancelPut()
    _, err = e.cli.Put(ctx, e.GenEtcdServiceKey(), e.GenEtcdServiceValue(), etcdctl.WithLease(e.lease.ID))
    if err != nil {
        ERROR_LOG("etcd put error %v", err)
        return
    }

    ch, err := e.cli.KeepAlive(context.TODO(), e.lease.ID)
    if err != nil {
        ERROR_LOG("etcd keep alive error %v", err)
        return
    }

    ka := <-ch
    INFO_LOG("etcd new lease id %v ttl %+v", e.lease.ID, ka)
}

func (e *EtcdDriver) ServiceDiscover() {

    timeCh := time.After(time.Microsecond)

    go func() {
        for !e.exit {
            select {
            case <- timeCh:
                ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
                defer cancel()
                rsp, err := e.cli.Get(ctx, e.GenEtcdWatchKey(), etcdctl.WithPrefix())
                if err != nil {
                    ERROR_LOG(fmt.Sprintf("etcd get error %v", err))
                    timeCh = time.After(time.Second * 5)
                    continue
                }

                for _, ev := range rsp.Kvs {
                    remoteInfo := strings.Split(string(ev.Key), "/")

                    namespace := remoteInfo[2]
                    service := remoteInfo[3]
                    ip := strings.Split(remoteInfo[4], ":")[0]
                    port, _ := strconv.Atoi(strings.Split(remoteInfo[4], ":")[1])
                    OnRemoteDiscover(namespace, service, ip, uint32(port))
                }

                // 10分钟全量拉一次
                timeCh = time.After(time.Second * 60 * 10)
            }
        }
    } ()
}

func (e *EtcdDriver) ServiceWatch() {
    DEBUG_LOG("etcd server watch start...")

    go func() {
        watchChan := e.cli.Watch(context.Background(), e.GenEtcdWatchKey(), etcdctl.WithPrefix())

        for wrsp := range watchChan {
            for _, ev := range wrsp.Events {
                DEBUG_LOG("etcd events %s %s", ev.Type, ev.Kv.Key)

                remoteInfo := strings.Split(string(ev.Kv.Key), "/")
                namespace := remoteInfo[2]
                service := remoteInfo[3]
                ip := strings.Split(remoteInfo[4], ":")[0]
                port, _ := strconv.Atoi(strings.Split(remoteInfo[4], ":")[1])

                switch ev.Type {
                case etcdctl.EventTypePut:
                    OnRemoteDiscover(namespace, service, ip, uint32(port))
                case etcdctl.EventTypeDelete:
                    OnRemoteDisappear(namespace, service, ip, uint32(port))
                default:
                    ERROR_LOG("unknow etcd event %s %s", ev.Type, ev.Kv.Key)
                }
            }
        }

        DEBUG_LOG("etcd server watch end...")
    } ()
}

func (e *EtcdDriver) GenEtcdServiceKey() string {
    return fmt.Sprintf("/ms/%s/%s/%s:%d", GlobalCfg.Namespace, GlobalCfg.Service, GetLocalIP(), GlobalCfg.Port)
}

func (e *EtcdDriver) GenEtcdServiceValue() string {
    e.leaseDV += 1
    return fmt.Sprintf("%d", e.leaseDV)
}

func (e *EtcdDriver) GenEtcdWatchKey() string {
    return fmt.Sprintf("/ms/%s", GlobalCfg.Namespace)
}

func (e *EtcdDriver) KeepAlive() {
    // etcd集群本身是高可靠且高可用的，应用层没必要做保活检测
}

var etcdDriver *EtcdDriver = nil

func CreateEtcdDriver() {
    var mode int8 = 0  // service regist

    if "ServiceGate" == GlobalCfg.Service || "ClientGate" == GlobalCfg.Service {
        mode = 1  // service discover
    }

    etcdDriver = &EtcdDriver{mode: mode}
}

func StartEtcdDriver() {
    etcdDriver.Start()
}

func StopEtcdDriver() {
    etcdDriver.Stop()
}
