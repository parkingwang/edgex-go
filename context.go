package edgex

import (
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
	NewEndpoint(opts EndpointOptions) Endpoint

	// 注册驱动
	NewDriver(opts DriverOptions) Driver

	// 注册管道
	NewPipeline(opts PipelineOptions) Pipeline

	// 返回等待通道
	WaitChan() <-chan os.Signal

	// 阻塞等待终止信号
	AwaitTerm()
}

////

func Run(handler func(ctx Context) error) {
	scoped := NewDefaultGlobalScoped("tcp://localhost:1883")
	ctx := newContext(scoped)
	log.Debug("启动Service")
	defer log.Debug("停止Service")
	if err := handler(ctx); nil != err {
		log.Error("Service出错: ", err)
	}
}

////

type context struct {
	scoped      *GlobalScoped
	serviceName string
	serviceId   string
}

// 加载配置
func (c *context) LoadConfig() map[string]interface{} {
	out := make(map[string]interface{})
	if _, err := toml.DecodeFile("application.toml", &out); nil != err {
		log.Error("读取配置文件(application.toml)出错: ", err)
	}
	return out
}

// 注册端点
func (c *context) NewEndpoint(opts EndpointOptions) Endpoint {
	checkContextInitialize(c)
	c.serviceName = "Endpoint"
	c.serviceId = opts.Id
	return &endpoint{
		scoped:        c.scoped,
		topic:         opts.Topic,
		id:            opts.Id,
		mqttTopicSend: topicOfEndpointSendQ(opts.Topic, opts.Id),
		mqttTopicRecv: topicOfEndpointRecvQ(opts.Topic, opts.Id),
		recvChan:      make(chan Frame, 2),
	}
}

// 注册驱动
func (c *context) NewDriver(opts DriverOptions) Driver {
	checkContextInitialize(c)
	c.serviceName = "Driver"
	c.serviceId = opts.Id
	return nil
}

// 注册管道
func (c *context) NewPipeline(opts PipelineOptions) Pipeline {
	checkContextInitialize(c)
	c.serviceName = "Pipeline"
	c.serviceId = opts.Id
	return nil
}

func (c *context) WaitChan() <-chan os.Signal {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	signal.Ignore(syscall.SIGPIPE)
	return sig
}

func (c *context) AwaitTerm() {
	<-c.WaitChan()
}

func newContext(global *GlobalScoped) Context {
	return &context{
		scoped: global,
	}
}

func checkContextInitialize(c *context) {
	if c.serviceName != "" {
		log.Panicf("Context已作为[%]服务使用", c.serviceName)
	}
}
