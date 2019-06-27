package edgex

//
// Author: 陈哈哈 chenyongjia@parkingwang.com, yoojiachen@gmail.com
//

const (
	NodeTypeEndpoint = "ENDPOINT"
	NodeTypeTrigger  = "TRIGGER"
)

// Inspect消息是将节点相关信息的描述
type Inspect struct {
	HostOS       string        `json:"hostOS"`
	HostArch     string        `json:"hostArch"`
	Vendor       string        `json:"vendor"`     // 品牌名称
	DriverName   string        `json:"driverName"` // 服务驱动类型名
	VirtualNodes []VirtualNode `json:"devices"`    // 虚拟设备节点列表
}

type VirtualNode struct {
	VirtualNodeName string                 `json:"name"`       // 虚拟设备名称，内部框架会自动填充所属节点名称作为前缀
	Desc            string                 `json:"desc"`       // 设备描述信息
	Type            string                 `json:"type"`       // 设备类型
	Virtual         bool                   `json:"virtual"`    // 是否为虚拟设备，即通过代理后转换的设备
	Command         string                 `json:"command"`    // 设备控制命令
	EventTopic      string                 `json:"eventTopic"` // 发送事件Topic
	Attrs           map[string]interface{} `json:"attrs"`      // 属性
	Address         string                 `json:"address"`    // 控制通讯地址
}
