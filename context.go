package edgex

import (
	"github.com/BurntSushi/toml"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

type Context interface {
	// 返回Log对象
	Log() *zap.SugaredLogger

	// 加载配置
	LoadConfig() map[string]interface{}

	// 创建Trigger
	NewTrigger(opts TriggerOptions) Trigger

	// 创建终端
	NewEndpoint(opts EndpointOptions) Endpoint

	// 创建驱动
	NewDriver(opts DriverOptions) Driver

	// 返回等待通道
	WaitChan() <-chan os.Signal

	// 阻塞等待终止信号
	AwaitTerm() error
}

////

func Run(handler func(ctx Context) error) {
	broker, ok := os.LookupEnv("MQTT.broker")
	if !ok {
		broker = "tcp://localhost:1883"
	}
	scoped := NewDefaultGlobalScoped(broker)
	ctx := newContext(scoped)
	log.Debug("启动Service")
	defer log.Debug("停止Service")
	if err := handler(ctx); nil != err {
		log.Error("Service出错: ", err)
	}
}

//// Context实现

type contextImpl struct {
	scoped      *GlobalScoped
	serviceName string
	serviceId   string
}

func (c *contextImpl) LoadConfig() map[string]interface{} {
	out := make(map[string]interface{})
	if _, err := toml.DecodeFile("application.toml", &out); nil != err {
		log.Error("读取配置文件(application.toml)出错: ", err)
	}
	return out
}

func (c *contextImpl) NewTrigger(opts TriggerOptions) Trigger {
	checkContextInitialize(c)
	checkRequired(opts.Name, "Trigger.Name MUST be specified")
	checkRequired(opts.Topic, "Trigger.Topic MUST be specified")
	c.serviceName = "Endpoint"
	c.serviceId = opts.Name
	return &trigger{
		scoped: c.scoped,
		topic:  opts.Topic,
		name:   opts.Name,
	}
}

func (c *contextImpl) NewEndpoint(opts EndpointOptions) Endpoint {
	checkContextInitialize(c)
	checkRequired(opts.Name, "Endpoint.Name MUST be specified")
	checkRequired(opts.Addr, "Endpoint.Addr MUST be specified")
	c.serviceName = "Endpoint"
	c.serviceId = opts.Addr
	return &endpoint{
		scoped:       c.scoped,
		endpointAddr: opts.Addr,
	}
}

func (c *contextImpl) NewDriver(opts DriverOptions) Driver {
	checkContextInitialize(c)
	checkRequired(opts.Name, "Endpoint.Name MUST be specified")
	checkRequireds(opts.Topics, "Endpoint.Topics MUST be specified")
	c.serviceName = "Driver"
	c.serviceId = opts.Name
	return &driver{
		scoped: c.scoped,
		name:   opts.Name,
		topics: opts.Topics,
	}
}

func (c *contextImpl) WaitChan() <-chan os.Signal {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	signal.Ignore(syscall.SIGPIPE)
	return sig
}

func (c *contextImpl) AwaitTerm() error {
	<-c.WaitChan()
	return nil
}

func (c *contextImpl) Log() *zap.SugaredLogger {
	return log
}

func newContext(global *GlobalScoped) Context {
	return &contextImpl{
		scoped: global,
	}
}

func checkContextInitialize(c *contextImpl) {
	if c.serviceName != "" {
		log.Panicf("Context已作为[%]服务使用", c.serviceName)
	}
}

func checkRequired(value, message string) {
	if "" == value {
		log.Panic(message)
	}
}

func checkRequireds(value []string, message string) {
	if 0 == len(value) {
		log.Panic(message)
	}
}
