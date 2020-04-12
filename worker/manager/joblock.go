package manager

import (
	"context"
	"fmt"

	"github.com/coreos/etcd/clientv3"
	"github.com/snow-flow/crontab/common"
)

// 分布式锁(TXN事务)
type JobLock struct {
	kv    clientv3.KV
	lease clientv3.Lease

	jobName    string
	cancalFunc context.CancelFunc // 终止自动续租
	leaseId    clientv3.LeaseID   // 租约ID
	isLocked   bool               // 是否上锁成功
}

// 初始化锁
func InitJobLock(jobName string, kv clientv3.KV, lease clientv3.Lease) (jobLock *JobLock) {
	return &JobLock{
		kv:      kv,
		lease:   lease,
		jobName: jobName,
	}
}

func (jobLock *JobLock) TryLock() (err error) {
	// 	1. 创建租约（5秒）
	grantResponse, err := jobLock.lease.Grant(context.TODO(), 5)
	if err != nil {
		fmt.Println(err)
		return err
	}

	cancel, cancelFunc := context.WithCancel(context.TODO())
	leaseID := grantResponse.ID

	// 取消并释放租约
	releaseLockFunc := func() {
		cancelFunc()
		jobLock.lease.Revoke(context.TODO(), leaseID)
	}

	// 2. 自动续租
	keepAliveResponsesChan, err := jobLock.lease.KeepAlive(cancel, leaseID)
	if err != nil {
		fmt.Println(err)
		releaseLockFunc()
		return err
	}

	go func() {
		for {
			select {
			case keepAliveResponses := <-keepAliveResponsesChan: // 自动续租应答
				if keepAliveResponses == nil {
					continue
				}

			}
		}
	}()

	// 3. 创建事务txn
	txn := jobLock.kv.Txn(context.TODO())
	lockKey := common.JOB_LOCK_DIR + jobLock.jobName

	// 4. 事务抢锁
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).
		Then(clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseID))).
		Else(clientv3.OpGet(lockKey))
	txnResponse, err := txn.Commit()

	if err != nil {
		fmt.Println(err)
		releaseLockFunc()
		return err
	}

	// 5. 成功返回，失败释放租约
	if !txnResponse.Succeeded { // 锁被占用
		releaseLockFunc()
		return common.ERR_LOCK_ALREADY_REQUIRED
	}

	// 6. 抢锁成功
	jobLock.leaseId = leaseID
	jobLock.cancalFunc = cancelFunc
	jobLock.isLocked = true

	return nil
}

// 释放锁
func (jobLock JobLock) UnLock() {
	if jobLock.isLocked {
		jobLock.cancalFunc()                                  // 取消自动续约的协程
		jobLock.lease.Revoke(context.TODO(), jobLock.leaseId) // 释放租约
	}
}
