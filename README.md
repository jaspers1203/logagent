# README

###1. 配置文件说明：

    可数组形式定义多文件夹，同时收集多日志信息；    
    支持多种不同数据源及多种上游接收目标，如本例日志数据源为文本文件格式，目标为KAFKA；    
    无具体需求，没有对日志行做域解析，以一行字符为基本单位；
      
        
    YAML格式配置文件appconfig.yml:
    
    agentType: FILE #当前代理器类别  FILE/TCP/KAFKA/。。。
    targetType: KAFKA #当前发送器类别  KAFKA/ES/。。。
    
    agentConfig:
      file:
        logDir:
          - d:\\logs\\info
          - d:\\logs\\error
        logLevel: INFO
    
      tcp:
        hostAddr: 192.168.128.136:7701    
    
    targetConfig:
      kafka:
        hostAddr: 192.168.128.136:9092
        topic:
      es:
        hostAddr: 192.168.128.136:9200
        index:
        

###2. 主函数调用
    初始化读取配置文件，按配置文件代理类型及上游接收类型，生成具体日志代理对象.


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
    
        switch cfg.AgentType {
        case logagent.AGENT_TYPE_FILE:
            agent = logagent.NewFileAgent(cfg)
            break
        case logagent.AGENT_TYPE_TCP:
            agent = logagent.NewTCPAgent(cfg)
            break
        case logagent.AGENT_TYPE_KAFKA:
            agent = logagent.NewKafkaAgent(cfg)
            break
        case "...":
            break
        }
        //启动代理器
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
    
    
###3. 定义LogAgentInterface接口，具体实例实现Run()函数


    //定义日志代理接口    
    type LogAgentInterface interface {
        Run()
    }

    3.1 FILE模式：
    
        //文件类日志代理器
        type FileAgent struct {
            filePath   []string
            fileMgr    *fileMgr
            watch      *fsnotify.Watcher
            cfgFile		*conf.AppConfig
        }
        
        //文件管理器
        type fileMgr struct {
            fileChan    chan string
            fileObjChan chan *fileObj
            msgChan     chan string
        }
        
        //文件处理对象
        type fileObj struct {
            //每个读取日志文件的对象
            tail     *tail.Tail
            offset   int64 //记录当前位置
            filename string
            sender   target.LogTargetInterface
        }
        
        //代理器实现接口
        func (fa *FileAgent) Run() {
        	fa.fileWatch()
        	fa.fileMgr.dispatch(fa.cfgFile)
        
        }
        
        生成FILE代理器对象
        func NewFileAgent(cfg *conf.AppConfig) LogAgentInterface {
        
        	fa := &FileAgent{
        		cfgFile: cfg,
        		fileMgr: &fileMgr{
        			fileChan: make(chan string, 1000),
        			msgChan:  make(chan string, 999999),
        		},
        	}
        	watch, err := fsnotify.NewWatcher()
        	if err != nil {
        		log.Fatalf("create dir watch error:%+v", err)
        		return nil
        	}
        	fa.watch = watch
        
        	for _, p := range cfg.AgentConfig.File.LogDir {
        		fa.filePath = append(fa.filePath, p)
        		rd, err := ioutil.ReadDir(p)
        		if err != nil {
        			continue
        		}
        		for _, fi := range rd {
        			if fi.IsDir() {
        			} else {
        				fa.fileMgr.fileChan <- fi.Name()
        			}
        		}
        	}
        
        	return fa
        }

        生成文件夹监控器，如有新文件产生自动添加到文件代理器文件处理Channel等待处理：
        func (fa *FileAgent) fileWatch() {
        	for _, fp := range fa.filePath {
        		fa.watch.Add(fp)

        		go func() {
        			for {
        				select {
        				case ev := <-fa.watch.Events:
        					if ev.Op&fsnotify.Create == fsnotify.Create {
        						fa.fileMgr.fileChan <- ev.Name
        						log.Println("创建文件 : ", ev.Name)
        					}
        					break
        				}
        			}
        		}()
        	}
        }

        //执行日志文件读取，为每个实际文件创建文件处理对象，含tail抓取器及Sender类型
        func (fm *fileMgr) dispatch(cfg *conf.AppConfig) {
        	//另起协程生成日志文件抓取对象
        	go func() {
        		for {
        			select {
        			case f := <-fm.fileChan:
        				tail, err := tail.TailFile(f, tail.Config{
        					ReOpen:    true,
        					Follow:    true,
        					Location:  &tail.SeekInfo{Offset: 0, Whence: 2},
        					MustExist: false,
        					Poll:      true,
        				})
        				if err != nil {
        					log.Printf("get tail file error:%+v\n", err)
        					continue
        				}
        				var sender target.LogTargetInterface
        				switch cfg.TargetType {
                        case TARGET_TYPE_ES:
                            sender = target.NewESTargetAgent()
                            break
                        case TARGET_TYPE_KAFKA:
                            sender = target.NewKafkaTargetAgent(cfg.TargetConfig.Kafka.HostAddr,cfg.TargetConfig.Kafka.Topic)
                            break
                        }
                                
                        if sender == nil {
                            log.Printf("invalid target sender instance")
                            continue
                        }
                        fileObj := &fileObj{
                            filename: f,
                            offset:   0,
                            tail:     tail,
                            sender:   sender,
                        }
        
                        fm.fileObjChan <- fileObj
                        break
                    }
        		}
        	}()
        
        	//执行日志读取
        	go func() {
        		for true {
        			select {
        			case fo := <-fm.fileObjChan:
        				go fo.processLog()
        				break
        			}
        		}
        	}()
        }

        /**
        	日志读取处理
         */
        func (fo *fileObj) processLog() {
        	for line := range fo.tail.Lines {
        		if line.Err != nil {
        			log.Printf("read line failed,err:%v", line.Err)
        			continue
        		}
        		msg := strings.TrimSpace(line.Text)
        		if len(msg) == 0 || msg[0] == '\n' {
        			continue
        		}

        		fo.sender.SendMessage(msg)
        	}
        }

    3.2 TCP模式：
    
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

#####4. 定义LogTargetInterface接口，具体发送者实现SendMessage()函数，日志代理对象与发送对象桥接
    
    
    //上游发送实现接口
    type LogTargetInterface interface {
        SendMessage(msg interface{})
    }
    
    4.1 KAFKA模式，NewKafkaTargetAgent函数创建异步Producer；    

        type KafkaTarget struct {
            producer sarama.AsyncProducer
            topic    string
        }

        func NewKafkaTargetAgent(host string, topic string) LogTargetInterface {
            var err error

            config := sarama.NewConfig()
            config.Producer.RequiredAcks = sarama.WaitForAll
            config.Producer.Partitioner = sarama.NewRandomPartitioner
            config.Producer.Return.Successes = true

            producer, err = sarama.NewAsyncProducer(strings.Split(host, ","), config)
            if err != nil {
                log.Printf("producer close,err:%s\n", err)
                return nil
            }

            return &KafkaTarget{
                producer: producer,
                topic:    topic,
            }
        }
        //实现接口
        func (kfk *KafkaTarget) SendMessage(inMsg interface{}) {
            var (
                m  string
                ok bool
            )
            if m, ok = inMsg.(string); !ok {

            }
            key, _ := time.Now().MarshalBinary()
            value, _ := json.Marshal(m)

            msg := &sarama.ProducerMessage{
                Topic: kfk.topic,
                Key:   sarama.ByteEncoder(key),
                Value: sarama.ByteEncoder(value),
            }

            producer.Input() <- msg

            select {
            case suc := <-producer.Successes():
                fmt.Printf("offset:%v,timestamp:%v\n", suc.Offset, suc.Timestamp.String())
            case fail := <-producer.Errors():
                fmt.Printf("err: %s\n", fail.Err.Error())
            }

        }
    
    
#####5. others