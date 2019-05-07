package edgex

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

type EndpointOptions struct {
	// 节点ID
	Id string
	// 消息的Topic
	Topic string
}

type DriverOptions struct {
	// 驱动ID
	Id string

	// 接受的Topic列表
	Topics []string
}

type PipelineOptions struct {
	// ID
	Id string

	// 接受的Topic列表
	Topics []string
}
