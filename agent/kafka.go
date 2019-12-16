//package agent

/**
 * @Author: syg
 * @Description: 
 * @File:  kafka.go
 * @Version: 1.0.0
 * @Date: 2019/12/16 10:20
 */

package agent

import "logagent/conf"

type KafkaAgent struct {
	cfgFile *conf.AppConfig
}

func NewKafkaAgent(cfg *conf.AppConfig) LogAgentInterface {

	return &KafkaAgent{
		cfgFile: cfg,
	}
}

func (ka *KafkaAgent) Run() {

}
