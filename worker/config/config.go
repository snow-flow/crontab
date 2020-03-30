package config

import (
	"encoding/json"
	"io/ioutil"
)

// 程序配置
type Config struct {
	EtcdEndPoints   []string `json:"etcd_end_points"`
	EtcdDialTimeout int      `json:"etcd_dial_timeout"`
}

var G_config *Config

func InitConfig(filename string) (err error) {
	// 	1. 读取配置文件
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	config := &Config{}
	// 	2. JSON反序列化
	err = json.Unmarshal(content, config)
	if err != nil {
		return err
	}

	// 3. 赋值单例
	G_config = config

	return nil
}
