package edgex

//
// Author: 陈哈哈 chenyongjia@parkingwang.com, yoojiachen@gmail.com
//

const (
	NodeTypeEndpoint = "ENDPOINT"
	NodeTypeTrigger  = "TRIGGER"
)

// 主节点属性模型
type MainNodeProperties struct {
	HostOS       string                   `json:"hostOS"`     // 系统
	HostArch     string                   `json:"hostArch"`   // CPU架构
	NodeType     string                   `json:"nodeType"`   // 节点类型
	NodeId       string                   `json:"nodeId"`     // 节点ID
	Vendor       string                   `json:"vendor"`     // 所属品牌名称
	ConnDriver   string                   `json:"connDriver"` // 通讯驱动名称
	VirtualNodes []*VirtualNodeProperties `json:"nodes"`      // 虚拟设备节点列表
}

// 虚拟节点属性模型
type VirtualNodeProperties struct {
	Uuid          string            `json:"uuid"`          // 自动生成的UUID；设备的唯一节点编号；
	VirtualId     string            `json:"virtualId"`     // 虚拟节点ID；须保证在单个节点内唯一性
	MajorId       string            `json:"majorId"`       // 设备主ID
	MinorId       string            `json:"minorId"`       // 设备次ID
	Description   string            `json:"description"`   // 设备描述信息
	Virtual       bool              `json:"virtual"`       // 是否为虚拟设备，即通过代理后转换的设备
	StateCommands map[string]string `json:"stateCommands"` // 各个状态下的控制指令
	Attrs         map[string]string `json:"attrs"`         // 其它属性列表
}

// 虚拟节点状态模型
type VirtualNodeState struct {
	NodeId    string                 `json:"nodeId"`    // NodeId
	Uuid      string                 `json:"uuid"`      // 自动生成的UUID；设备的唯一节点编号；
	VirtualId string                 `json:"virtualId"` // 虚拟节点ID；须保证在单个节点内唯一性
	MajorId   string                 `json:"majorId"`   // 设备主ID
	MinorId   string                 `json:"minorId"`   // 设备次ID
	State     string                 `json:"state"`     // 设备状态
	Values    map[string]interface{} `json:"values"`    // 设备状态数值
}
