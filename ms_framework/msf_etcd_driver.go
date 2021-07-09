package ms_framework

import (
    // etcd "go.etcd.io/etcd/client/v3"
    etcdctl "go.etcd.io/etcd/clientv3"  // need grpc@v1.26.0
    "context"
    "time"
    "fmt"
    "strings"
    "strconv"
)

type EtcdDriver struct {
    mode     int8  // 0 - service regist  1 - service discover
    cli     *etcdctl.Client
    lease   *etcdctl.LeaseGrantResponse
}

func (e *EtcdDriver) Start() {
    cli, err := etcdctl.New(etcdctl.Config{
        Endpoints:   GlobalCfg.Etcd,
        DialTimeout: 5 * time.Second,
    })

    if err != nil {
        ERROR_LOG("etcd clientv3 new error %v %v", err, GlobalCfg.Etcd)
        return
    }

    // DEBUG_LOG("etcd driver client: %+v", cli)

    e.cli = cli

    if 0 == e.mode {
        e.ServiceRegist()
    } else if 1 == e.mode {
        e.ServiceDiscover()
        e.ServiceWatch()
    }
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
}

func (e *EtcdDriver) ServiceRegist() {
    // minimum lease ttl is 2 second
    // lease, err := e.cli.Grant(context.TODO(), 2)
    ctx, cancelGrant := context.WithTimeout(context.Background(), 5 * time.Second)
    defer cancelGrant()
    lease, err := e.cli.Grant(ctx, 2)
    if err != nil {
        panic(fmt.Sprintf("etcd grant error %v", err))
    }

    e.lease = lease

    ctx, cancelPut := context.WithTimeout(context.Background(), 5 * time.Second)
    defer cancelPut()
    _, err = e.cli.Put(ctx, e.GenEtcdServiceKey(), e.GenEtcdServiceValue(), etcdctl.WithLease(e.lease.ID))
    if err != nil {
        panic(fmt.Sprintf("etcd put error %v", err))
    }

    ch, err := e.cli.KeepAlive(context.TODO(), e.lease.ID)
    if err != nil {
        panic(fmt.Sprintf("etcd keep alive error %v", err))
    }

    ka := <-ch
    INFO_LOG("etcd lease id %v ttl %v", e.lease.ID, ka.TTL)
}

func (e *EtcdDriver) ServiceDiscover() {
    ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
    defer cancel()
    rsp, err := e.cli.Get(ctx, e.GenEtcdWatchKey(), etcdctl.WithPrefix())
    if err != nil {
        panic(fmt.Sprintf("etcd get error %v", err))
    }

    for _, ev := range rsp.Kvs {
        remoteInfo := strings.Split(string(ev.Key), "/")

        namespace := remoteInfo[2]
        service := remoteInfo[3]
        ip := strings.Split(remoteInfo[4], ":")[0]
        port, _ := strconv.Atoi(strings.Split(remoteInfo[4], ":")[1])
        OnRemoteDiscover(namespace, service, ip, uint32(port))
    }
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
    return "nil"
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
