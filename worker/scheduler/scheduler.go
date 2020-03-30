package scheduler

import "github.com/snow-flow/crontab/common"

// 任务调度
type Scheduler struct {
	jobEventChan chan *common.JobEvent // etcd 任务事件队列
}

// 调度协程
func (scheduler *Scheduler) scheduleLoop() {
	// 	定时任务
}

// 初始化调度器
var G_scheduler *Scheduler

func initScheduler() (err error) {
	G_scheduler = &Scheduler{jobEventChan: make(chan *common.JobEvent, 1000)}

	// 启动调度协程
	go G_scheduler.scheduleLoop()
	return nil
}
