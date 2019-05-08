package edgex

import (
	"github.com/BurntSushi/toml"
	"github.com/yoojia/edgex/util"
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

type context struct {
	scoped      *GlobalScoped
	serviceName string
	serviceId   string
}

func (c *context) LoadConfig() map[string]interface{} {
	out := make(map[string]interface{})
	if _, err := toml.DecodeFile("application.toml", &out); nil != err {
		log.Error("读取配置文件(application.toml)出错: ", err)
	}
	return out
}

func (c *context) NewTrigger(opts TriggerOptions) Trigger {
	checkContextInitialize(c)
	c.serviceName = "Endpoint"
	c.serviceId = opts.Name
	return &trigger{
		scoped: c.scoped,
		topic:  opts.Topic,
		name:   opts.Name,
	}
}

func (c *context) NewEndpoint(opts EndpointOptions) Endpoint {
	checkContextInitialize(c)
	c.serviceName = "Endpoint"
	c.serviceId = opts.Id
	return &endpoint{
		scoped:           c.scoped,
		endpointId:       opts.Id,
		mqttTopicRequest: topicOfEndpointRequestQ(opts.Id),
		mqttTopicReply:   topicOfEndpointReplyQ(opts.Id),
	}
}

func (c *context) NewDriver(opts DriverOptions) Driver {
	checkContextInitialize(c)
	c.serviceName = "Driver"
	c.serviceId = opts.Name
	return &driver{
		scoped:  c.scoped,
		name:    opts.Name,
		topics:  opts.Topics,
		replies: new(util.MapX),
	}
}

func (c *context) WaitChan() <-chan os.Signal {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	signal.Ignore(syscall.SIGPIPE)
	return sig
}

func (c *context) AwaitTerm() error {
	<-c.WaitChan()
	return nil
}

func (c *context) Log() *zap.SugaredLogger {
	return log
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
