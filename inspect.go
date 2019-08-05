package edgex

//
// Author: 陈哈哈 chenyongjia@parkingwang.com, yoojiachen@gmail.com
//

const (
	NodeTypeEndpoint = "ENDPOINT"
	NodeTypeTrigger  = "TRIGGER"
)

// MainNodeInfo 是主节点相关信息的描述
type MainNodeInfo struct {
	HostOS       string             `json:"hostOS"`     // 系统
	HostArch     string             `json:"hostArch"`   // CPU架构
	NodeType     string             `json:"nodeType"`   // 节点类型
	NodeId       string             `json:"nodeId"`     // 节点ID
	Vendor       string             `json:"vendor"`     // 所属品牌名称
	ConnDriver   string             `json:"connDriver"` // 通讯驱动名称
	VirtualNodes []*VirtualNodeInfo `json:"nodes"`      // 虚拟设备节点列表
}

// 虚拟节点信息
type VirtualNodeInfo struct {
	Uuid      string                 `json:"uuid"`      // 自动生成的UUID；设备的唯一节点编号；
	VirtualId string                 `json:"virtualId"` // 虚拟节点ID；须保证在单个节点内唯一性
	MajorId   string                 `json:"majorId"`   // 设备主ID
	MinorId   string                 `json:"minorId"`   // 设备次ID
	Desc      string                 `json:"desc"`      // 设备描述信息
	Virtual   bool                   `json:"virtual"`   // 是否为虚拟设备，即通过代理后转换的设备
	Commands  string                 `json:"commands"`  // 设备控制命令
	Attrs     map[string]interface{} `json:"attrs"`     // 其它属性
}
