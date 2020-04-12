package common

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gorhill/cronexpr"
)

// 定时任务
type Job struct {
	Name     string `json:"name"`      // 任务名
	Command  string `json:"command"`   // shell命令
	CronExpr string `json:"cron_expr"` // cron表达式
}

// 任务调度计划
type JobSchedulePlan struct {
	Job      *Job                 // 要调度的任务信息
	Expr     *cronexpr.Expression // 解析好的cronexpr表达式
	NextTime time.Time            // 下次调度时间
}

// 任务执行状态
type JobExecuteInfo struct {
	Job        *Job               // 任务信息
	PlanTime   time.Time          // 理论上的调度时间
	RealTime   time.Time          // 实际的调度时间
	CancelCtx  context.Context    // 任务command的context
	CancelFunc context.CancelFunc // 用于取消command执行的cancel函数
}

// HTTP接口应答
type Response struct {
	Errno   int         `json:"errno"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// 变化事件
type JobEvent struct {
	EventType int // save, delete
	Job       *Job
}

// 任务执行结果
type JobExecuteResult struct {
	ExecuteInfo *JobExecuteInfo // 执行状态
	Output      []byte          // 脚本输出
	Err         error           // 脚本错误原因
	StartTime   time.Time       // 启动时间
	EndTime     time.Time       // 结束时间
}

// 任务执行日志
type JobLog struct {
	JobName      string `bson:"job_name"`      // 任务名字
	Command      string `bson:"command"`       // 脚本命令
	Err          string `bson:"err"`           // 错误原因
	Output       string `bson:"output"`        // 脚本输出
	PlanTime     int64  `bson:"plan_time"`     // 计划开始时间
	ScheduleTime int64  `bson:"schedule_time"` //实际调度时间
	StartTime    int64  `bson:"start_time"`    // 任务执行开始时间
	EndTime      int64  `bson:"end_time"`      // 任务执行结束时间
}

// 日志批次
type LogBatch struct {
	Logs []interface{} // 多条日志
}

// 应答方法
func BuildResponse(errno int, message string, data interface{}) (resp []byte, err error) {
	response := Response{
		Errno:   errno,
		Message: message,
		Data:    data,
	}

	// 	 序列化
	return json.Marshal(response)
}

// 反序列化Job
func UnpackJob(value []byte) (ret *Job, err error) {
	job := &Job{}
	err = json.Unmarshal(value, job)
	if err != nil {
		return nil, err
	}

	return job, nil
}

// 从 etcd 的key中提取任务名
func ExtractJobName(jobkey string) string {
	return strings.TrimPrefix(jobkey, JOB_SAVE_DIR)
}

// 从 etcd 的/cron/killer/key 中提取任务名
func ExtractKillerName(killerkey string) string {
	return strings.TrimPrefix(killerkey, JOB_KILLER_DIR)
}

// 任务变化事件有两种：1）更新任务 2）删除任务
func BuildJobEvent(eventType int, job *Job) (jobEvent *JobEvent) {
	return &JobEvent{
		EventType: eventType,
		Job:       job,
	}
}

// 构造任务执行计划
func BuildJobSchedulePlan(job *Job) (jobSchedulePlan *JobSchedulePlan) {
	expression, err := cronexpr.Parse(job.CronExpr)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	// 	生成任务调度计划对象
	return &JobSchedulePlan{
		Job:      job,
		Expr:     expression,
		NextTime: expression.Next(time.Now()),
	}
}

func BuildJobExecuteInfo(jobSchedulePlan *JobSchedulePlan) (jobExecuteInfo *JobExecuteInfo) {
	jobExecuteInfo = &JobExecuteInfo{
		Job:      jobSchedulePlan.Job,
		PlanTime: jobSchedulePlan.NextTime,
		RealTime: time.Now(),
	}

	jobExecuteInfo.CancelCtx, jobExecuteInfo.CancelFunc = context.WithCancel(context.TODO())

	return jobExecuteInfo
}
