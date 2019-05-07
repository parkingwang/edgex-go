package edgex

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

type Driver interface {
	// 消息输入通道
	Inputs() <-chan Frame
	// 消息输出通道
	Outputs() chan<- Frame
	// 发起一个消息请求，并获取响应消息。如果过程中发生错误，返回错误消息
	Request(req Frame) (resp Frame, err error)
}
