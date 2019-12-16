# README

1. 配置文件说明：
    可以数组形式定义多文件夹，同时收集多文件日志信息；    
    支持多种不同数据源及多种上游接收目标，如本例日志数据源为文本文件格式，目标为KAFKA；    
    无具体需求，没有对日志行做域映射，以一行字符为基本单位；

    
    logConfig:    
      logDir:     
        - 'd:\\logs\\info'
        - 'd:\\logs\\error'
      logLevel: INFO
      
    agentConfig:
      source:
        name: FILE
      target:
        name: KAFKA
        host: 192.168.128.136:9092
        index:
        

2. 定义LogAgentInterface接口，具体实例实现Run()函数


    //定义日志代理接口    
    type LogAgentInterface interface {
        Run()
    }

    2.1 FILE模式：
    
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
        
        	for _, p := range cfg.LogConfig.LogDir {
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
        func (fm *fileMgr) dispatch(targetType string) {
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
        				switch targetType {
        				case ES:
        					sender = target.NewESTargetAgent()
        					break
        				case KAFKA:
        					sender = target.NewKafkaTargetAgent()
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

    2.2 TCP模式：

3. 定义LogTargetInterface接口，具体发送者实现SendMessage()函数，日志代理对象与发送对象桥接

    3.1 KAFKA模式，NewKafkaTargetAgent函数创建异步Producer；
    

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
