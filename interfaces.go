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
	// GenerateEventId 返回消息事件ID
	GenerateEventId() int64

	// NewMessage 创建基于节点的消息对象
	NewMessage(groupId, majorId, minorId string, body []byte, eventId int64) Message
}

// 发布State/Properties消息
type NeedProperties interface {
	// 发送节点属性消息
	PublishNodeProperties(properties MainNodeProperties)

	// 发送节点状态消息
	PublishNodeState(state VirtualNodeState)
}
