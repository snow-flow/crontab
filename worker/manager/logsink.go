package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/snow-flow/crontab/common"
	"github.com/snow-flow/crontab/worker/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// mongodb存储日志
type LogSink struct {
	client         *mongo.Client
	collection     *mongo.Collection
	logChan        chan *common.JobLog
	autoCommitChan chan *common.LogBatch
}

var G_logSink *LogSink

// 批量写入日志
func (logSink *LogSink) saveLogs(batch *common.LogBatch) {
	logSink.collection.InsertMany(context.Background(), batch.Logs)
}

// 日志存储协程
func (logSink *LogSink) writeLoop() {
	var logBatch *common.LogBatch
	var timeoutBatch *common.LogBatch
	var commitTimer *time.Timer
	for {
		select {
		case log := <-logSink.logChan:
			// 批量插入
			if logBatch == nil {
				logBatch = &common.LogBatch{}
				// 	超时自动提交
				commitTimer = time.AfterFunc(time.Duration(config.G_config.JobLogCommitTimeout)*time.Millisecond,
					func(batch *common.LogBatch) func() {
						return func() {
							// 	发送超时通知，不要直接提交batch
							logSink.autoCommitChan <- batch
						}

					}(logBatch),
				)

			}

			// 	把日志追加到日志中
			logBatch.Logs = append(logBatch.Logs, log)

			// 	如果批次满了，就立即发送
			if len(logBatch.Logs) >= config.G_config.JobLogBatchSize {
				logSink.saveLogs(logBatch)
				logBatch = nil
				// 	取消定时器
				commitTimer.Stop()
			}

		case timeoutBatch = <-logSink.autoCommitChan: // 过期的批次
			if timeoutBatch != logBatch {
				continue
			}
			logSink.saveLogs(timeoutBatch)
			logBatch = nil
		}
	}
}

func InitLogSink() (err error) {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 	选择DB和Collection
	G_logSink = &LogSink{
		client:         client,
		collection:     client.Database("cron").Collection("log"),
		logChan:        make(chan *common.JobLog, 1000),
		autoCommitChan: make(chan *common.LogBatch, 1000),
	}

	// 启动mongo存储协程
	go G_logSink.writeLoop()

	return nil
}

// 发送日志
func (logSink *LogSink) Append(jobLog *common.JobLog) {
	select {
	case logSink.logChan <- jobLog:
	default: // 队列满了就丢弃
	}
}
