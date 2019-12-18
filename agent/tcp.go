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

//TCP日志代理器
type TCPAgent struct {
	cfgFile *conf.AppConfig
}

//每个Socket连接实际处理对象
type TCPPipe struct {
	conn    *net.TCPConn
	msgChan chan string
	sender  target.LogTargetInterface
}

func NewTCPAgent(cfg *conf.AppConfig) LogAgentInterface {

	//var err error
	return &TCPAgent{
		cfgFile: cfg,
		//sender:  sender,
	}
}

//代理器启动
func (ta *TCPAgent) Run() {
	//var tcpAddr *net.TCPAddr
	var sender target.LogTargetInterface

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

		switch ta.cfgFile.TargetType {
		case TARGET_TYPE_KAFKA:
			host := ta.cfgFile.TargetConfig.Kafka.HostAddr
			topic := ta.cfgFile.TargetConfig.Kafka.Topic
			sender = target.NewKafkaTargetAgent(host, topic)
			break
		case TARGET_TYPE_ES:
			break
		}

		if sender == nil {
			log.Printf("create sender error\n")
			continue
		}

		tp := &TCPPipe{
			conn:    tcpConn,
			sender:  sender,
			msgChan: make(chan string, 999999),
		}

		go tp.tcpPipe()
	}

}

//具体处理连接过程方法
func (tp *TCPPipe) tcpPipe() {
	defer func() {
		log.Printf("connection disconnect %s", tp.conn.RemoteAddr())
		tp.conn.Close()
	}()

	//另起线程处理Channel中已接收消息
	go func() {
		for {
			select {
			case msg := <-tp.msgChan:
				tp.sender.SendMessage(msg)
			}
		}
	}()

	reader := bufio.NewReader(tp.conn)
	//循环接收消息
	for {
		msg, err := reader.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		tp.msgChan <- msg
	}
}
