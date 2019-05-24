package edgex

//
// Author: 陈哈哈 chenyongjia@parkingwang.com, yoojiachen@gmail.com
//

const (
	DeviceTypeEndpoint = "ENDPOINT"
	DeviceTypeTrigger  = "TRIGGER"
)

// Inspect消息是将节点相关信息的描述
type Inspect struct {
	HostOS     string   `json:"hostOs,omitempty"`
	HostArch   string   `json:"hostArch,omitempty"`
	Vendor     string   `json:"vendor,omitempty"`     // 品牌名称
	DriverName string   `json:"driverName,omitempty"` // 服务驱动类型名
	Devices    []Device `json:"devices,omitempty"`    // 设备列表
}

type Device struct {
	Name       string `json:"name,omitempty"`       // 设备名称
	Desc       string `json:"desc,omitempty"`       // 设备描述信息
	Type       string `json:"type,omitempty"`       // 设备类型
	Virtual    bool   `json:"virtual,omitempty"`    // 是否为虚拟设备，即通过代理后转换的设备
	Command    string `json:"command,omitempty"`    // 设备控制命令
	Address    string `json:"address,omitempty"`    // 控制通讯地址
	EventTopic string `json:"eventTopic,omitempty"` // 发送事件Topic
}
