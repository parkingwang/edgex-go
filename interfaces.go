package edgex

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

type NeedLifecycle interface {
	Startup()
	Shutdown()
}

type NeedNodeName interface {
	NodeName() string
}

// 创建消息接口
type NeedMessages interface {
	// NextMessageSequenceId 返回内部消息流水号
	NextMessageSequenceId() uint32

	// NextMessageByVirtualId 返回指定虚拟节点ID，基于内部流水号的消息对象
	NextMessageByVirtualId(virtualNodeId string, body []byte) Message

	// NextMessageBySourceUuid
	NextMessageBySourceUuid(sourceUuid string, body []byte) Message
}
