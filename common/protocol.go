package common

import "encoding/json"

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
