package edgex

//
// Author: 陈哈哈 chenyongjia@parkingwang.com, yoojiachen@gmail.com
//

const (
	NodeTypeEndpoint = "ENDPOINT"
	NodeTypeTrigger  = "TRIGGER"
)

// EdgeNode消息是节点相关信息的描述
type EdgeNode struct {
	HostOS       string        `json:"hostOS"`     // 系统
	HostArch     string        `json:"hostArch"`   // CPU架构
	NodeType     string        `json:"nodeType"`   // 节点类型
	Vendor       string        `json:"vendor"`     // 所属品牌名称
	DriverName   string        `json:"driverName"` // 驱动类型名
	VirtualNodes []VirtualNode `json:"nodes"`      // 虚拟设备节点列表
}

type VirtualNode struct {
	Major      string                 `json:"major"`   // 主ID
	Minor      string                 `json:"minor"`   // 次ID
	Desc       string                 `json:"desc"`    // 设备描述信息
	Type       string                 `json:"type"`    // 设备类型
	Virtual    bool                   `json:"virtual"` // 是否为虚拟设备，即通过代理后转换的设备
	RpcCommand string                 `json:"command"` // 设备控制命令
	RpcAddress string                 `json:"address"` // 控制通讯地址
	Attrs      map[string]interface{} `json:"attrs"`   // 其它属性
}
