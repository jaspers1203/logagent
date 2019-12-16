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

type AppConfig struct {
	LogConfig LogConfig	`yaml:"logConfig"`
	AgentConfig AgentConfig `yaml:"agentConfig"`
}

type LogConfig struct {
	LogLevel string `yaml:"logLevel"`
	LogDir   []string `yaml:"logDir,flow"`
}

type AgentConfig struct {
	Source struct {
		Name string `yaml:"name"`
	} `yaml:"source"`
	Target  struct {
		Name string `yaml:"name"`
		Host string `yaml:"host"`
		Index string `yaml:"index"`
		Topic string `yaml:"topic"`
	} `yaml:"target"`
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
	if err!=nil{
		fmt.Printf("read config file error:%+v\n",err)
		return err
	}
	err = yaml.Unmarshal(file, c)
	if err!=nil{
		fmt.Printf("Unmarshal config file error:%+v\n",err)
		return err
	}
}
