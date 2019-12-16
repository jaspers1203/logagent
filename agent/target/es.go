//package target

/**
 * @Author: syg
 * @Description: 
 * @File:  es.go
 * @Version: 1.0.0
 * @Date: 2019/12/16 10:22
 */

package target

type ESTarget struct {}

func NewESTargetAgent() LogTargetInterface {
	return &ESTarget{}
}

func (es *ESTarget) SendMessage(msg interface{}) {

}
