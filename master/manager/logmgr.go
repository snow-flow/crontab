package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/snow-flow/crontab/common"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// mongodb日志管理
type LogMgr struct {
	client     *mongo.Client
	collection *mongo.Collection
}

var G_logMgr *LogMgr

func InitLogMgr() (err error) {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 	选择DB和Collection
	G_logMgr = &LogMgr{
		client:     client,
		collection: client.Database("cron").Collection("log"),
	}

	return nil
}

// 查看任务日志
func (logMgr *LogMgr) ListLog(name string, skip int, limit int) (logArr []*common.JobLog, err error) {
	filter := &common.JobLogFilter{JobName: name}
	sort := &common.SortLogByStartTime{SortOrder: -1}

	logArr = make([]*common.JobLog, 0)

	cursor, err := logMgr.collection.Find(context.Background(), filter,
		options.Find().SetSort(sort),
		options.Find().SetSkip(int64(skip)),
		options.Find().SetLimit(int64(limit)),
	)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		jobLog := &common.JobLog{}
		err = cursor.Decode(jobLog)
		if err != nil {
			continue
		}

		logArr = append(logArr, jobLog)
	}

	return logArr, nil
}
