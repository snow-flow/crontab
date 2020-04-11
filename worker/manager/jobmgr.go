package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/snow-flow/crontab/common"
	"github.com/snow-flow/crontab/worker/config"
	"github.com/snow-flow/crontab/worker/scheduler"

	"github.com/coreos/etcd/clientv3"
)

// 任务管理器
type JobMgr struct {
	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
	watcher clientv3.Watcher
}

var G_jobMgr *JobMgr

// 初始化管理器
func InitJobMgr() (err error) {
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
	watcher := clientv3.NewWatcher(client)

	// 	赋值单例
	G_jobMgr = &JobMgr{
		client:  client,
		kv:      kv,
		lease:   lease,
		watcher: watcher,
	}

	// 启动监听
	G_jobMgr.watchJobs()

	return nil
}

// 监听任务变化
func (m *JobMgr) watchJobs() (err error) {
	fmt.Println("----watching jobs")
	var jobEvent *common.JobEvent
	// 	1. get一下/cron/jobs/目录下的所有任务，并且获知当前集群的revision
	getResponse, err := m.kv.Get(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix())
	if err != nil {
		fmt.Print(err)
		return err
	}

	for _, kvpair := range getResponse.Kvs {
		job, err := common.UnpackJob(kvpair.Value)
		if err != nil {
			fmt.Println(err)
			return err
		}
		jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
		fmt.Printf("%v", *jobEvent)
		// 把这个job同步给scheduler（调度协程）
		scheduler.G_scheduler.PushJobEvent(jobEvent)
	}

	// 	2. 从该revision向后监听变化事件
	go func() {
		// 	从GET时刻的后续版本开始监听变化
		watchStartRevision := getResponse.Header.Revision + 1
		// 	监听/cron/jobs/目录的后续变化
		watchChan := m.watcher.Watch(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithRev(watchStartRevision), clientv3.WithPrefix())
		// 处理监听事件
		for watchResponse := range watchChan {
			for _, watchEvent := range watchResponse.Events {
				switch watchEvent.Type {
				case mvccpb.PUT: // 任务保存事件
					// TODO：反序列化Job，推送更新事件给scheduler
					job, err := common.UnpackJob(watchEvent.Kv.Value)
					if err != nil {
						continue
					}
					// 	构建一个更新Event事件
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)

				case mvccpb.DELETE: // 任务被删除了
					jobName := common.ExtractJobName(string(watchEvent.Kv.Key))

					// 构造一个删除EVENT
					job := &common.Job{Name: jobName}
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_DELETE, job)

				}
				// fmt.Println(*jobEvent)
				//  推送删除更新事件给scheduler
				scheduler.G_scheduler.PushJobEvent(jobEvent)
			}
		}
	}()

	return nil
}
