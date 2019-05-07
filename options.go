package edgex

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

type EndpointOptions struct {
	// 节点名称
	Name string

	// 消息的Topic
	Topic string
}

type DriverOptions struct {
	// 驱动名称
	Name string

	// 接受的Topic列表
	Topics []string
}

type PipelineOptions struct {
	// 名称
	Name string

	// 接受的Topic列表
	Topics []string
}
