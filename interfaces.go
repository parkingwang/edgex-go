package edgex

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// 生命周期接口
type NeedLifecycle interface {
	Startup()
	Shutdown()
}

// 节点ID接口
type NeedNodeId interface {
	NodeId() string
}

// 创建消息接口
type NeedMessages interface {
	// NextMessageSequenceId 返回内部消息流水号
	NextMessageSequenceId() uint32

	// NextMessageBy 根据指令VirtualId，使用内部NodeId，创建基于内部流水号的消息对象
	NextMessageBy(virtualId string, body []byte) Message

	// NextMessageOf 根据完整VirtualNodeId，创建基于内部流水号的消息对象
	NextMessageOf(virtualNodeId string, body []byte) Message
}

type NeedInspect interface {
	// 发布Inspect消息
	PublishInspect(node MainNodeInfo)
}
