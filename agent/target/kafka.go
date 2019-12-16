//package target

/**
 * @Author: syg
 * @Description: 
 * @File:  kafka.go
 * @Version: 1.0.0
 * @Date: 2019/12/16 10:22
 */

package target

import (
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"log"
	"strings"
	"time"
)

var producer sarama.AsyncProducer

func init() {

}

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
