package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/snow-flow/crontab/common"

	"github.com/snow-flow/crontab/master/config"

	"github.com/coreos/etcd/clientv3"
)

// 任务管理器
type JobMgr struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

var G_jobMgr *JobMgr

// 初始化管理器
func InitJobMgr() (err error) {
	// 	初始化配置
	conf := clientv3.Config{
		Endpoints:   config.G_config.EtcdEndPoints,
		DialTimeout: time.Duration(config.G_config.EtcdDialTimeout) * time.Microsecond,
	}

	// 	建立客户端
	client, err := clientv3.New(conf)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 	得到KV和Lease的API子集
	kv := clientv3.NewKV(client)
	lease := clientv3.NewLease(client)

	// 	赋值单例
	G_jobMgr = &JobMgr{
		client: client,
		kv:     kv,
		lease:  lease,
	}

	return nil
}

// 保存任务
func (m *JobMgr) SaveJob(job *common.Job) (oldjob *common.Job, err error) {
	// 	把任务保存到/cron/jobs/任务名 = json
	jobkey := common.JOB_SAVE_DIR + job.Name

	jobval, err := json.Marshal(job)
	if err != nil {
		fmt.Println(err)
		return
	}
	// 	保存到ETCD
	putResponse, err := m.kv.Put(context.TODO(), jobkey, string(jobval), clientv3.WithPrevKV())
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// 	如果是更新，那么返回旧值
	oldjobret := &common.Job{}

	if putResponse.PrevKv != nil {
		err := json.Unmarshal(putResponse.PrevKv.Value, oldjobret)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		return oldjobret, nil
	}

	return nil, nil

}

// 删除任务
func (m *JobMgr) DeleteJob(name string) (oldjob *common.Job, err error) {
	jobkey := common.JOB_SAVE_DIR + name
	deleteResponse, err := m.kv.Delete(context.TODO(), jobkey, clientv3.WithPrevKV())
	if err != nil {
		fmt.Println(err)
		return
	}

	oldjobret := &common.Job{}
	// 返回被删除的任务信息
	if len(deleteResponse.PrevKvs) != 0 {
		err = json.Unmarshal(deleteResponse.PrevKvs[0].Value, &oldjobret)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		return oldjobret, nil
	}

	return nil, nil
}

// 获取所有任务
func (m *JobMgr) ListJobs() (jobList []*common.Job, err error) {
	getResponse, err := m.kv.Get(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix())
	if err != nil {
		fmt.Println(err)
		return
	}

	jobs := make([]*common.Job, 0)
	for _, kv := range getResponse.Kvs {
		job := &common.Job{}
		err := json.Unmarshal(kv.Value, job)
		if err != nil {
			continue
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// 杀死任务
func (m *JobMgr) KillJob(name string) (err error) {
	killerkey := common.JOB_KILLER_DIR + name
	// 更新一下key=/cron/killer/任务名  worker监听变化
	// 	让worker监听到一次PUT操作，创建一个租约让其过期即可
	grantResponse, err := m.lease.Grant(context.TODO(), 1)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 	租约ID
	leaseID := grantResponse.ID

	// 	设置killer标记
	_, err = m.kv.Put(context.TODO(), killerkey, "", clientv3.WithLease(leaseID))
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
