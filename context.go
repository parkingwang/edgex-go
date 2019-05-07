package edgex

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
	"os/signal"
	"syscall"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

type Context interface {
	// 加载配置
	LoadConfig() map[string]interface{}

	// 注册端点
	NewEndpoint(opts EndpointOptions) (Endpoint, NodeInfo)

	// 注册驱动
	NewDriver(opts DriverOptions) (Driver, NodeInfo)

	// 注册管道
	NewPipeline(opts PipelineOptions) (Pipeline, NodeInfo)

	// 返回等待通道
	WaitChan() <-chan os.Signal

	// 阻塞等待终止信号
	AwaitTerm()
}

type NodeInfo struct {
	Topic string
	Uuid  string
}

////

func Run(handler func(ctx Context) error) {
	global := &Global{
		MqttBroker: "tcp://localhost:1883",
	}
	ctx := newContext(global)
	log.Debug("启动Service")
	defer log.Debug("停止Service")
	if err := handler(ctx); nil != err {
		log.Error("Service出错: ", err)
	}
}

////

type ctx struct {
	global       *Global
	serviceName  string
	instanceName string
	info         NodeInfo
}

// 加载配置
func (c *ctx) LoadConfig() map[string]interface{} {
	out := make(map[string]interface{})
	if _, err := toml.DecodeFile("application.toml", &out); nil != err {
		log.Error("读取配置文件(application.toml)出错: ", err)
	}
	return out
}

// 注册端点
func (c *ctx) NewEndpoint(opts EndpointOptions) (Endpoint, NodeInfo) {
	c.serviceName = "Endpoint"
	c.instanceName = opts.Name
	c.info = NodeInfo{
		Topic: opts.Topic,
		Uuid:  fmt.Sprintf("EdgeX-Endpoint-%s", opts.Name),
	}
	worker := &endpoint{
		global: c.global,
		info:   c.info,
	}
	return worker, c.info
}

// 注册驱动
func (c *ctx) NewDriver(opts DriverOptions) (Driver, NodeInfo) {
	return nil, c.info
}

// 注册管道
func (c *ctx) NewPipeline(opts PipelineOptions) (Pipeline, NodeInfo) {
	return nil, c.info
}

func (c *ctx) WaitChan() <-chan os.Signal {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	signal.Ignore(syscall.SIGPIPE)
	return sig
}

func (c *ctx) AwaitTerm() {
	<-c.WaitChan()
}

func newContext(global *Global) Context {
	return &ctx{
		global: global,
	}
}
