package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/snow-flow/crontab/master/manager"

	"github.com/snow-flow/crontab/common"
	"github.com/snow-flow/crontab/master/config"
)

// 任务的HTTP接口
type ApiServer struct {
	httpServer *http.Server
}

var G_apiServer *ApiServer

// 保存任务接口
// POST job={"name":"job1", "command":"echo hello", "cronExpr":"* * * * *"}
func handleJobSave(w http.ResponseWriter, r *http.Request) {
	// 	保存到ETCD中
	var job common.Job
	// 	1. 解析POST表单
	err := r.ParseForm()
	if err != nil {
		fmt.Println(err)
		return
	}

	// 	2. 获取表单中的job字段
	jobJson := r.PostForm.Get("job")

	// 3. 反序列化job
	err = json.Unmarshal([]byte(jobJson), &job)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 	 4. 保存到ETCD
	saveJob, err := manager.G_jobMgr.SaveJob(&job)
	if err != nil {
		fmt.Println(err)
		return
	}

	response, err := common.BuildResponse(0, "success", saveJob)
	if err == nil {
		w.Write(response)
	}

}

// 删除任务接口
// POST /job/delete job={"name":"job1", "command":"echo hello", "cronExpr":"* * * * *"}
func handleJobDelete(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Println(err)
		return
	}

	// 	删除的任务名
	name := r.PostForm.Get("name")

	// 	去删除任务
	oldjob, err := manager.G_jobMgr.DeleteJob(name)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 正常应答
	response, err := common.BuildResponse(0, "success", oldjob)
	if err == nil {
		w.Write(response)
		return
	}

	resp, _ := common.BuildResponse(-1, err.Error(), nil)
	w.Write(resp)

}

// 获取任务接口
func handleJobList(w http.ResponseWriter, r *http.Request) {
	jobList, err := manager.G_jobMgr.ListJobs()
	if err != nil {
		fmt.Println(err)
		return
	}

	// 正常应答
	response, err := common.BuildResponse(0, "success", jobList)
	if err == nil {
		w.Write(response)
		return
	}

	resp, _ := common.BuildResponse(-1, err.Error(), nil)
	w.Write(resp)

}

// 强制杀死某个任务
// POST /job/kill name=job1
func handleJobKill(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Println(err)
		return
	}

	// 	杀死的任务名
	name := r.PostForm.Get("name")
	err = manager.G_jobMgr.KillJob(name)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 正常应答
	response, err := common.BuildResponse(0, "success", nil)
	if err == nil {
		w.Write(response)
		return
	}

	resp, _ := common.BuildResponse(-1, err.Error(), nil)
	w.Write(resp)

}

// 查询任务日志
func handleJobLog(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Println(err)
		return
	}

	// /job/log?name=job1&skip=0&limit=10
	name := r.Form.Get("name")
	skip, _ := strconv.Atoi(r.Form.Get("skip"))
	limit, _ := strconv.Atoi(r.Form.Get("limit"))

	listLog, err := manager.G_logMgr.ListLog(name, skip, limit)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 正常应答
	response, err := common.BuildResponse(0, "success", listLog)
	if err == nil {
		w.Write(response)
		return
	}

	resp, _ := common.BuildResponse(-1, err.Error(), nil)
	w.Write(resp)

}

// 初始化服务
func InitApiServer() (err error) {
	// 	配置路由
	mux := http.NewServeMux()
	mux.HandleFunc("/job/save", handleJobSave)
	mux.HandleFunc("/job/delete", handleJobDelete)
	mux.HandleFunc("/job/list", handleJobList)
	mux.HandleFunc("/job/kill", handleJobKill)
	mux.HandleFunc("/job/log", handleJobLog)

	// /index.html
	// 静态文件目录
	staticDir := http.Dir(config.G_config.WebRoot)
	staticHandler := http.FileServer(staticDir)
	mux.Handle("/", http.StripPrefix("/", staticHandler))

	// 启动TCP监听
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(config.G_config.ApiPort))
	if err != nil {
		return err
	}

	// 创建一个HTTP服务
	httpServer := &http.Server{
		ReadTimeout:  time.Duration(config.G_config.ApiReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(config.G_config.ApiWriteTimeout) * time.Millisecond,
		Handler:      mux,
	}

	// 赋值单例
	G_apiServer = &ApiServer{httpServer: httpServer}

	// 启动服务端
	go httpServer.Serve(listener)

	return nil
}
