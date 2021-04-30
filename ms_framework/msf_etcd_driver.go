package ms_framework

import (
	etcd "go.etcd.io/etcd/client/v3"
	"context"
	"time"
)

type EtcdDriver struct {
	cli 	*etcd.Client
}

func (e *EtcdDriver) New() {
	cli, err := etcd.New(etcd.Config{
		Endpoints:   GlobalCfg.Etcd,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		ERROR_LOG("etcd clientv3 new error %v %v", err, GlobalCfg.Etcd)
		return
	}

	defer cli.Close()

	INFO_LOG("dail etcd %v %v", GlobalCfg.Etcd, cli)
	ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
	_, err = cli.Put(ctx, "youjun_foo", "youjun_bar")
	cancel()
	if err != nil {
		ERROR_LOG("etcd put error %v", err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10 * time.Second)
	resp, err := cli.Get(ctx, "youjun_foo")
	cancel()
	if err != nil {
		ERROR_LOG("etcd get error %v", err)
	}
	for _, ev := range resp.Kvs {
		DEBUG_LOG("%s : %s\n", ev.Key, ev.Value)
	}
}