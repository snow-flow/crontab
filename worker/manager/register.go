package manager

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/snow-flow/crontab/common"
	"github.com/snow-flow/crontab/worker/config"
)

// 注册节点到etcd：/cron/worker/ip
type Register struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease

	localIP string // 本机IP
}

var G_register *Register

// 获取本机网卡IP
func getLocalIP() (ipv4 string, err error) {

	// 	获取所有网卡
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	// 	取第一个非Lo的网卡IP
	for _, addr := range addrs {
		if ipNet, isIpNet := addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}

	return "", common.ERR_NO_LOCAL_IP_FOUND
}

// 注册到/cron/workers/IP，并自动续租
func (register *Register) keepOnline() {
	for {
		// 注册路径
		regKey := common.JOB_WORKER_DIR + register.localIP

		cancelCtx, cancelFunc := context.WithCancel(context.TODO())

		// 	创建租约
		grantResponse, err := register.lease.Grant(context.TODO(), 10)
		if err != nil {
			time.Sleep(1 * time.Second)
			cancelFunc()
			continue
		}

		// 	自动续租
		keepAliveChan, err := register.lease.KeepAlive(context.TODO(), grantResponse.ID)
		if err != nil {
			time.Sleep(1 * time.Second)
			cancelFunc()
			continue
		}

		// 	注册到etcd
		_, err = register.kv.Put(cancelCtx, regKey, "", clientv3.WithLease(grantResponse.ID))
		if err != nil {
			time.Sleep(1 * time.Second)
			cancelFunc()
			continue
		}

		// 	处理续租应答
		for {
			select {
			case keepAliveResp := <-keepAliveChan:
				if keepAliveResp == nil { // 续租失败
					time.Sleep(1 * time.Second)
					cancelFunc()
					continue
				}
			}
		}
	}

}

func InitRegister() (err error) {
	// 	初始化配置
	conf := clientv3.Config{
		Endpoints:   config.G_config.EtcdEndPoints,
		DialTimeout: time.Duration(config.G_config.EtcdDialTimeout) * time.Microsecond,
	}

	client, err := clientv3.New(conf)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 获取本机IP
	localIP, err := getLocalIP()
	if err != nil {
		return err
	}

	// 	得到KV和Lease的API子集
	kv := clientv3.NewKV(client)
	lease := clientv3.NewLease(client)
	// watcher := clientv3.NewWatcher(client)
	G_register = &Register{
		client:  client,
		kv:      kv,
		lease:   lease,
		localIP: localIP,
	}

	// 服务注册
	go G_register.keepOnline()
	return nil
}
