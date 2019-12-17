//package agent

/**
 * @Author: syg
 * @Description: 
 * @File:  tcp.go
 * @Version: 1.0.0
 * @Date: 2019/12/16 10:20
 */

package agent

import (
	"bufio"
	"io"
	"log"
	"logagent/agent/target"
	"logagent/conf"
	"net"
	"os"
)

type TCPAgent struct {
	cfgFile *conf.AppConfig
	sender  target.LogTargetInterface
}

func NewTCPAgent(cfg *conf.AppConfig) LogAgentInterface {
	var sender target.LogTargetInterface

	switch cfg.TargetType {
	case TARGET_TYPE_KAFKA:
		host := cfg.TargetConfig.Kafka.HostAddr
		topic := cfg.TargetConfig.Kafka.Topic
		sender = target.NewKafkaTargetAgent(host, topic)
		break
	case TARGET_TYPE_ES:
		break
	}

	return &TCPAgent{
		cfgFile: cfg,
		sender:  sender,
	}
}

func (ta *TCPAgent) Run() {
	//var tcpAddr *net.TCPAddr
	tcpAddr, err := net.ResolveTCPAddr("tcp", ta.cfgFile.AgentConfig.TCP.HostAddr)
	if err != nil {
		log.Printf("resolve tcp addr error:%+v\n", err)
		os.Exit(-1)
	}

	tcpListener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Printf("listen tcp error:%+v\n", err)
		os.Exit(-1)
	}
	defer tcpListener.Close()

	//循环接收客户端连接，创建新协程处理请求
	for {
		tcpConn, err := tcpListener.AcceptTCP()
		if err != nil {
			log.Printf("receive tcpConn error:%+v\n", err)
			continue
		}

		go ta.tcpPipe(tcpConn)
	}

}

//具体处理连接过程方法
func (ta *TCPAgent) tcpPipe(conn *net.TCPConn) {
	defer func() {
		log.Printf("connection disconnect %s", conn.RemoteAddr())
		conn.Close()
	}()

	reader := bufio.NewReader(conn)
	//循环接收消息并发送
	for {
		msg,err:= reader.ReadString('\n')
		if err!=nil||err==io.EOF{
			break
		}
		ta.sender.SendMessage(msg)
	}
}
