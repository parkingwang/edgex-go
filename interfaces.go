package edgex

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

type Lifecycle interface {
	Startup()
	Shutdown()
}

type NodeName interface {
	NodeName() string
}
