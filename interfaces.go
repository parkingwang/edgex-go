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

	// NextMessageByVirtualId 返回指定虚拟节点ID，基于内部流水号的消息对象
	NextMessageByVirtualId(virtualNodeId string, body []byte) Message

	// NextMessageBySourceUuid 返回指定源UUID，基于内部流水号的消息对象
	NextMessageBySourceUuid(sourceUuid string, body []byte) Message
}
