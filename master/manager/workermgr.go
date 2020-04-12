package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/snow-flow/crontab/common"
	"github.com/snow-flow/crontab/master/config"
)

type WorkerMgr struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

var G_workerMgr *WorkerMgr

// 获取在线worker列表
func (workerMgr WorkerMgr) ListWorkers() (workerArr []string, err error) {

	getResponse, err := workerMgr.kv.Get(context.TODO(), common.JOB_WORKER_DIR, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	worker := make([]string, 0)

	// 	解析每个节点的IP
	for _, kv := range getResponse.Kvs {
		workerIP := common.ExtractWorkerIP(string(kv.Key))
		worker = append(worker, workerIP)
	}

	return worker, nil
}

func InitWorkerMgr() (err error) {

	// 	初始化配置
	conf := clientv3.Config{
		Endpoints:   config.G_config.EtcdEndPoints,
		DialTimeout: time.Duration(config.G_config.EtcdDialTimeout) * time.Microsecond,
	}

	client, err := clientv3.New(conf)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 	得到KV和Lease的API子集
	kv := clientv3.NewKV(client)
	lease := clientv3.NewLease(client)

	G_workerMgr = &WorkerMgr{
		client: client,
		kv:     kv,
		lease:  lease,
	}

	return nil
}
