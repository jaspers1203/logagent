//package agent

/**
 * @Author: syg
 * @Description: 
 * @File:  tcp.go
 * @Version: 1.0.0
 * @Date: 2019/12/16 10:20
 */

package agent

import "logagent/conf"

type TCPAgent struct {
	cfgFile *conf.AppConfig
}

func NewTCPAgent(cfg *conf.AppConfig) LogAgentInterface{

	return &TCPAgent{
		cfgFile:cfg,
	}
}

func (ta *TCPAgent) Run(){

}
