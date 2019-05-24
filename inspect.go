package edgex

//
// Author: 陈哈哈 chenyongjia@parkingwang.com, yoojiachen@gmail.com
//

// Inspect消息是将节点相关信息的描述
type Inspect struct {
	OS      string   `json:"os,omitempty"`
	Arch    string   `json:"arch,omitempty"`
	Devices []Device `json:"devices,omitempty"`
}

type Device struct {
	// DeviceName
	Name string `json:"name,omitempty"`
	// Is virtual
	Virtual bool `json:"virtual,omitempty"`
	// Description of device
	Desc string `json:"desc,omitempty"`
	// Exec command
	Command string `json:"command,omitempty"`
}
