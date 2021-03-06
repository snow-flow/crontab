package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/snow-flow/crontab/worker/config"
	"github.com/snow-flow/crontab/worker/manager"
)

// 配置文件路径
var confFile string

// 解析命令行参数
func initArgs() {
	flag.StringVar(&confFile, "config", "main/worker.json", "config worker.json")
	flag.Parse()
}

func main() {
	initArgs()
	// 加载配置
	err := config.InitConfig(confFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 服务注册
	err = manager.InitRegister()
	if err != nil {
		fmt.Println(err)
		return
	}

	// 启动日志协程
	err = manager.InitLogSink()
	if err != nil {
		fmt.Println(err)
		return
	}

	// 启动执行器
	err = manager.InitExecutor()
	if err != nil {
		fmt.Println(err)
		return
	}

	// 启动调度器
	err = manager.InitScheduler()
	if err != nil {
		fmt.Println(err)
		return
	}

	// 任务管理器
	err = manager.InitJobMgr()
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		time.Sleep(1 * time.Second)
	}
}
