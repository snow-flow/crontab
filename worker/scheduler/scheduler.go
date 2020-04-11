package scheduler

import (
	"fmt"
	"time"

	"github.com/snow-flow/crontab/common"
)

// 任务调度
type Scheduler struct {
	jobEventChan      chan *common.JobEvent              // etcd 任务事件队列
	jobPlanTable      map[string]*common.JobSchedulePlan // 任务调度计划表
	jobExecutingTable map[string]*common.JobExecuteInfo  // 任务执行表
	jobResultChan     chan *common.JobExecuteResult      // 任务结果队列
}

// 处理任务事件
func (scheduler *Scheduler) handleJobEvent(jobEvent *common.JobEvent) {
	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE: // 保存任务事件
		jobSchedulePlan := common.BuildJobSchedulePlan(jobEvent.Job)
		if jobSchedulePlan != nil {
			scheduler.jobPlanTable[jobEvent.Job.Name] = jobSchedulePlan
		}
	case common.JOB_EVENT_DELETE: // 删除任务事件
		if _, jobExisted := scheduler.jobPlanTable[jobEvent.Job.Name]; jobExisted {
			delete(scheduler.jobPlanTable, jobEvent.Job.Name)
		}
	}
}

// 尝试执行任务
func (scheduler *Scheduler) TryStartJob(jobPlan *common.JobSchedulePlan) {
	// 	调度和执行是两件事情
	var jobExecuteInfo *common.JobExecuteInfo
	var jobExecuting bool

	// 执行的任务可能运行很久，1分钟调度60次，但是只能执行一次，防止并发
	if jobExecuteInfo, jobExecuting = scheduler.jobExecutingTable[jobPlan.Job.Name]; jobExecuting {
		fmt.Println("尚未退出，跳过执行：", jobPlan.Job.Name)
		return
	}

	// 构建执行状态信息
	jobExecuteInfo = common.BuildJobExecuteInfo(jobPlan)

	// 保存执行状态
	scheduler.jobExecutingTable[jobPlan.Job.Name] = jobExecuteInfo

	// 执行任务
	fmt.Println("执行任务：", jobExecuteInfo.Job.Name, jobExecuteInfo.PlanTime, jobExecuteInfo.RealTime)
	G_executor.ExecuteJob(jobExecuteInfo)
}

// 重新计算任务调度状态
func (scheduler *Scheduler) TrySchedule() (scheduleAfter time.Duration) {
	var nearTime *time.Time

	now := time.Now()

	// 如果任务表为空的话，随便睡眠多久
	if len(scheduler.jobPlanTable) == 0 {
		return 1 * time.Second
	}

	// 1. 遍历所有任务
	for _, schedulePlan := range scheduler.jobPlanTable {
		if schedulePlan.NextTime.Before(now) || schedulePlan.NextTime.Equal(now) {
			// 	 TODO: 尝试执行任务
			fmt.Println("执行任务：", schedulePlan.Job.Name)
			scheduler.TryStartJob(schedulePlan)
			schedulePlan.NextTime = schedulePlan.Expr.Next(now) // 更新下次执行时间
		}

		// 统计一个最近要过期的任务时间
		if nearTime == nil || schedulePlan.NextTime.Before(*nearTime) {
			nearTime = &schedulePlan.NextTime
		}
	}

	return (*nearTime).Sub(now)

}

// 处理任务结果
func (scheduler Scheduler) handleJobResult(result *common.JobExecuteResult) {
	delete(scheduler.jobExecutingTable, result.ExecuteInfo.Job.Name)
	fmt.Println("任务执行完成：", result.ExecuteInfo.Job.Name, string(result.Output), result.Err)
}

// 调度协程
func (scheduler *Scheduler) scheduleLoop() {
	// 初始化一次
	scheduleAfter := scheduler.TrySchedule()

	// 调度的延时定时器
	scheduleTimer := time.NewTimer(scheduleAfter)

	// 	定时任务common.Job
	for {
		select {
		case jobEvent := <-scheduler.jobEventChan: // 监听任务变化事件
			// 	对内存中维护的任务列表做增删改查
			scheduler.handleJobEvent(jobEvent)
		case <-scheduleTimer.C: // 最近的任务到期了
		case jobResult := <-scheduler.jobResultChan: // 监听任务执行结果
			scheduler.handleJobResult(jobResult)
		}

		// 调度一次任务
		scheduleAfter = scheduler.TrySchedule()
		// 重置调度间隔
		scheduleTimer.Reset(scheduleAfter)
	}
}

// 推送任务变化事件
func (scheduler *Scheduler) PushJobEvent(jobEvent *common.JobEvent) {
	scheduler.jobEventChan <- jobEvent
}

var G_scheduler *Scheduler

// 初始化调度器
func InitScheduler() (err error) {
	G_scheduler = &Scheduler{
		jobEventChan:      make(chan *common.JobEvent, 1000),
		jobPlanTable:      make(map[string]*common.JobSchedulePlan),
		jobExecutingTable: make(map[string]*common.JobExecuteInfo),
		jobResultChan:     make(chan *common.JobExecuteResult, 1000),
	}

	// 启动调度协程
	go G_scheduler.scheduleLoop()
	return nil
}

// 回传任务执行结果
func (scheduler *Scheduler) PushJobResult(jobResult *common.JobExecuteResult) {
	scheduler.jobResultChan <- jobResult
}
