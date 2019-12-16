//package target

/**
 * @Author: syg
 * @Description: 
 * @File:  itarget.go
 * @Version: 1.0.0
 * @Date: 2019/12/16 10:48
 */

package target

//上游发送实现接口
type LogTargetInterface interface {
	SendMessage(msg interface{})
}