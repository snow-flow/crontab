package manager

import (
	"math/rand"
	"os/exec"
	"time"

	"github.com/snow-flow/crontab/common"
)

// 任务执行器
type Executor struct {
}

var G_executor *Executor

// 执行一个任务
func (executor *Executor) ExecuteJob(info *common.JobExecuteInfo) {
	go func() {
		// 任务结果
		result := &common.JobExecuteResult{
			ExecuteInfo: info,
			Output:      make([]byte, 0),
		}

		// 首先获取分布式锁
		jobLock := G_jobMgr.CreatJobLock(info.Job.Name)

		// 记录任务开始时间
		result.StartTime = time.Now()

		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

		err := jobLock.TryLock()
		defer jobLock.UnLock()

		if err != nil {
			result.Err = err
			result.EndTime = time.Now()
		} else {
			// 上锁成功后，重置任务启动时间
			result.StartTime = time.Now()

			// 	执行shell命令
			cmd := exec.CommandContext(info.CancelCtx, "/bin/bash", "-c", info.Job.Command)

			// 	执行并捕获输出
			output, err := cmd.CombinedOutput()

			// 记录任务结束时间
			result.EndTime = time.Now()
			result.Output = output
			result.Err = err
		}

		// 	任务执行完成后，把执行的结果返回给Scheduler, Scheduler会从executingTable中删除掉执行记录
		G_scheduler.PushJobResult(result)

	}()

}

// 初始化执行器
func InitExecutor() (err error) {

	G_executor = &Executor{}

	return nil
}
