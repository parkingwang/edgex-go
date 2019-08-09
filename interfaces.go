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
type NeedAccessNodeId interface {
	NodeId() string
}

// 创建消息接口
type NeedCreateMessages interface {
	// NextMessageSequenceId 返回内部消息流水号
	NextMessageSequenceId() uint32

	// NextMessageBy 根据指令VirtualId，使用内部NodeId，创建基于内部流水号的消息对象
	NextMessageBy(virtualId string, body []byte) Message

	// NextMessageOf 根据完整VirtualNodeId，创建基于内部流水号的消息对象
	NextMessageOf(virtualNodeId string, body []byte) Message
}

// 发布Inspect消息
type NeedInspectProperties interface {
	// 发送节点属性消息
	PublishNodeProperties(properties MainNodeProperties)

	// 发送节点状态消息
	PublishNodeState(state VirtualNodeState)
}
