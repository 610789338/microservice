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
	mode 	int8  // 0 - service regist  1 - service discover
	cli 	*etcdctl.Client
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

	e.cli = cli
	INFO_LOG("connect etcd %v success", GlobalCfg.Etcd)

	if 0 == e.mode {
		e.ServiceRegist()
	} else if 1 == e.mode {
		e.ServiceDiscover()
	}
}

func (e *EtcdDriver) Stop() {
	INFO_LOG("etcd driver stop...")

	if e.lease != nil {
		_, err := e.cli.Revoke(context.TODO(), e.lease.ID)
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
	lease, err := e.cli.Grant(context.TODO(), 2)
	if err != nil {
		ERROR_LOG("etcd grant error %v", err)
		return
	}

	e.lease = lease

	ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
	_, err = e.cli.Put(ctx, e.GenEtcdServiceKey(), e.GenEtcdServiceValue(), etcdctl.WithLease(e.lease.ID))
	cancel()
	if err != nil {
		ERROR_LOG("etcd put error %v", err)
		return
	}

	ch, err := e.cli.KeepAlive(context.TODO(), e.lease.ID)
	if err != nil {
		ERROR_LOG("etcd keep alive error %v", err)
	}

	ka := <-ch
	INFO_LOG("etcd lease id %v ttl %v", e.lease.ID, ka.TTL)
}

func (e *EtcdDriver) ServiceDiscover() {
	ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
	rsp, err := e.cli.Get(ctx, e.GenEtcdWatchKey(), etcdctl.WithPrefix())
	cancel()
	if err != nil {
		ERROR_LOG("etcd get error %v", err)
		return
	}

	for _, ev := range rsp.Kvs {
		remoteInfo := strings.Split(string(ev.Key), "/")

		// DEBUG_LOG("etcd discover: %v", remoteInfo)

		namespace := remoteInfo[2]
		service := remoteInfo[3]
		ip := strings.Split(remoteInfo[4], ":")[0]
		port, _ := strconv.Atoi(strings.Split(remoteInfo[4], ":")[1])
		OnRemoteDiscover(namespace, service, ip, uint32(port))
	}

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
				WARN_LOG("unknow etcd event %s %s", ev.Type, ev.Kv.Key)
			}
		}
	}

	DEBUG_LOG("etcd server discover end...")
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

var etcdDriver *EtcdDriver = nil

func CreateEtcdDriver() {
	var mode int8 = 0  // service regist

	if "ServiceGate" == GlobalCfg.Service || "ClientGate" == GlobalCfg.Service {
		mode = 1  // service discover
	}

	etcdDriver = &EtcdDriver{mode: mode}
}

func StartEtcdDriver() {
	go etcdDriver.Start()
}

func StopEtcdDriver() {
	etcdDriver.Stop()
}
