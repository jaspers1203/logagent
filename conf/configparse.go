//package conf

/**
 * @Author: syg
 * @Description: 
 * @File:  configparse.go
 * @Version: 1.0.0
 * @Date: 2019/12/16 9:32
 */

package conf

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"runtime"
)

//整体配置文件结构体
type AppConfig struct {
	AgentType    string       `yaml:"agentType"`
	TargetType   string       `yaml:"targetType"`
	AgentConfig  AgentConfig  `yaml:"agentConfig"`
	TargetConfig TargetConfig `yaml:"targetConfig"`
}

//代理器配置
type AgentConfig struct {
	File struct {
		LogLevel string   `yaml:"logLevel"`
		LogDir   []string `yaml:"logDir,flow"`
	} `yaml:"file"`
	TCP struct {
		HostAddr string `yaml:"hostAddr"`
	} `yaml:"tcp"`
}

//发送器配置
type TargetConfig struct {
	Kafka struct {
		HostAddr string `yaml:"hostAddr"`
		Topic    string `yaml:"topic"`
	} `yaml:"kafka"`
	ES struct {
		HostAddr string `yaml:"hostAddr"`
		Index    string `yaml:"index"`
	} `yaml:"es"`
}

func (c *AppConfig) LoadConfig() error {
	var filePath string
	if runtime.GOOS == "windows" {
		filePath, _ = os.Getwd()
		filePath = filePath + "\\conf\\appconfig.yml"
	} else if runtime.GOOS == "linux" {
		filePath, _ = os.Getwd()
		filePath = filePath + "/conf/appconfig.yml"
	}

	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("read config file error:%+v\n", err)
		return err
	}
	err = yaml.Unmarshal(file, c)
	if err != nil {
		fmt.Printf("Unmarshal config file error:%+v\n", err)
		return err
	}

	return nil
}
