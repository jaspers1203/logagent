//package logagent

/**
 * @Author: syg
 * @Description: 
 * @File:  main.go
 * @Version: 1.0.0
 * @Date: 2019/12/16 12:27
 */

package main

import (
	"gopkg.in/yaml.v2"
	"log"
	logagent "logagent/agent"
	"logagent/conf"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var cfg *conf.AppConfig

func init() {
	cfg = &conf.AppConfig{}
	err := cfg.LoadConfig()
	if err != nil {
		os.Exit(-1)
	}
}

func main() {
	var wg sync.WaitGroup
	var agent logagent.LogAgentInterface

	switch cfg.AgentConfig.Source.Name {
	case "FILE":
		agent = logagent.NewFileAgent(cfg)
		break
	case "TCP":
		agent = logagent.NewTCPAgent(cfg)
		break
	case "KAFKA":
		agent = logagent.NewKafkaAgent(cfg)
		break
	case "...":
		break
	}

	agent.Run()

	//监控退出程序信号
	wg.Add(1)
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, os.Kill, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			<-s
			log.Println("log agent terminated")
			wg.Done()
		}
	}()

	wg.Wait()
}

//测试配置文件
func testCfgFile() {
	projectPath, _ := os.Getwd()
	log.Printf("project path:%s\n", projectPath)

	cfg := &conf.AppConfig{}
	cfg.LoadConfig()

	out, _ := yaml.Marshal(cfg)
	log.Printf("cfg file:\n%+v\n", string(out))
}
