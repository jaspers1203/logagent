//package agent

/**
 * @Author: syg
 * @Description: 
 * @File:  file.go
 * @Version: 1.0.0
 * @Date: 2019/12/16 10:20
 */

package agent

import (
	"github.com/fsnotify/fsnotify"
	"github.com/hpcloud/tail"
	"io/ioutil"
	"log"
	"logagent/agent/target"
	"logagent/conf"
	"strings"
)

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

func (fa *FileAgent) Run() {
	fa.fileWatch()
	fa.fileMgr.dispatch(fa.cfgFile)

}

//监控文件夹文件增删
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
				switch cfg.AgentConfig.Target.Name {
				case ES:
					sender = target.NewESTargetAgent()
					break
				case KAFKA:
					sender = target.NewKafkaTargetAgent(cfg.AgentConfig.Target.Host,cfg.AgentConfig.Target.Topic)
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
