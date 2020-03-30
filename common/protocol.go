package common

import (
	"encoding/json"
	"strings"
)

// 定时任务
type Job struct {
	Name     string `json:"name"`      // 任务名
	Command  string `json:"command"`   // shell命令
	CronExpr string `json:"cron_expr"` // cron表达式
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
	job       *Job
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

// 从ETCD的key中提取任务名
func ExtractJobName(jobkey string) string {
	return strings.TrimPrefix(jobkey, JOB_SAVE_DIR)
}

// 任务变化事件有两种：1）更新任务 2）删除任务
func BuildJobEvent(eventType int, job *Job) (jobEvent *JobEvent) {
	return &JobEvent{
		EventType: eventType,
		job:       job,
	}
}
