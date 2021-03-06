package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/snow-flow/crontab/common"
	"github.com/snow-flow/crontab/worker/config"
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

	// 启动任务监听
	G_jobMgr.watchJobs()

	// 启动Killer监听
	G_jobMgr.watchKiller()

	return nil
}

// 创建任务执行锁
func (m JobMgr) CreatJobLock(jobName string) (jobLock *JobLock) {
	// 返回一把锁
	return InitJobLock(jobName, m.kv, m.lease)
}

// 监听任务变化
func (m *JobMgr) watchJobs() (err error) {
	fmt.Println("========== 开始监听任务变化 =========")
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
		// 把这个job同步给scheduler（调度协程）
		G_scheduler.PushJobEvent(jobEvent)
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
				G_scheduler.PushJobEvent(jobEvent)
			}
		}
	}()

	return nil
}

// 监听强杀任务变化
func (m *JobMgr) watchKiller() {
	var jobEvent *common.JobEvent
	go func() {
		// 	监听/cron/killer/目录的后续变化
		watchChan := m.watcher.Watch(context.TODO(), common.JOB_KILLER_DIR, clientv3.WithPrefix())
		// 处理监听事件
		for watchResponse := range watchChan {
			for _, watchEvent := range watchResponse.Events {
				switch watchEvent.Type {
				case mvccpb.PUT: // 杀死某个任务事件
					killerName := common.ExtractKillerName(string(watchEvent.Kv.Key))
					fmt.Println("正在杀死：" + killerName)
					job := &common.Job{
						Name: killerName,
					}
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_KILL, job)
					//  推送更新事件给scheduler
					G_scheduler.PushJobEvent(jobEvent)
				case mvccpb.DELETE: // killer标记过期，被自动删除

				}

			}
		}
	}()

}
