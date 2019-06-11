package edgex

import (
	"errors"
	"fmt"
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

	// 当设置verbose环境变量
	LogIfVerbose(fn func(log *zap.SugaredLogger))

	// 加载配置
	LoadConfig() map[string]interface{}

	// 创建Trigger
	NewTrigger(opts TriggerOptions) Trigger

	// 创建终端
	NewEndpoint(opts EndpointOptions) Endpoint

	// 创建驱动
	NewDriver(opts DriverOptions) Driver

	// 返回等待通道
	TermChan() <-chan os.Signal

	// 阻塞等待终止信号
	TermAwait() error
}

const (
	EnvKeyBroker     = "EDGEX_MQTT_BROKER"
	EnvKeyConfig     = "EDGEX_CONFIG"
	EnvKeyLogVerbose = "EDGEX_LOG_VERBOSE"

	MqttBrokerDefault = "tcp://mqtt-broker.edgex.io:1883"

	DefaultConfName = "application.toml"
	DefaultConfFile = "/etc/edgex/application.toml"
)

var (
	ErrConfigNotExist = errors.New("config not exists")
)

////

func Run(handler func(ctx Context) error) {
	broker, ok := os.LookupEnv(EnvKeyBroker)
	if !ok {
		broker = MqttBrokerDefault
	}
	scoped := NewDefaultGlobalScoped(broker)
	ctx := newContext(scoped)
	log.Debug("启动Service")
	defer log.Debug("停止Service")
	if err := handler(ctx); nil != err {
		log.Error("Service出错: ", err)
	}
}

func CreateContext(scoped *GlobalScoped) Context {
	return newContext(scoped)
}

func CreateDefaultContext() Context {
	broker, ok := os.LookupEnv(EnvKeyBroker)
	if !ok {
		broker = MqttBrokerDefault
	}
	return CreateContext(NewDefaultGlobalScoped(broker))
}

//// Context实现

type implContext struct {
	scoped      *GlobalScoped
	serviceType string
	serviceName string
	logVerbose  bool
}

func (c *implContext) LoadConfig() map[string]interface{} {
	searchConfig := func(files ...string) (f string, err error) {
		for _, file := range files {
			if _, err := os.Stat(file); nil == err {
				return file, nil
			}
		}
		return "", ErrConfigNotExist
	}
	config := make(map[string]interface{})
	file, err := searchConfig(DefaultConfName, DefaultConfFile, os.Getenv(EnvKeyConfig))
	if nil != err {
		log.Panic("未设置任何配置文件")
	} else {
		log.Debug("加载配置文件：", file)
	}
	if _, err := toml.DecodeFile(file, &config); nil != err {
		log.Error(fmt.Sprintf("读取配置文件(%s)出错: ", file), err)
	}
	return config
}

func (c *implContext) NewTrigger(opts TriggerOptions) Trigger {
	c.checkInit()
	checkRequired(opts.Name, "Trigger.Name MUST be specified")
	checkRequired(opts.Topic, "Trigger.Topic MUST be specified")
	checkRequired(opts.InspectFunc, "Trigger.InspectFunc MUST be specified")
	c.serviceType = "Trigger"
	c.serviceName = checkNameFormat(opts.Name)
	return &implTrigger{
		scoped:      c.scoped,
		topic:       opts.Topic,
		name:        opts.Name,
		inspectFunc: opts.InspectFunc,
	}
}

func (c *implContext) NewEndpoint(opts EndpointOptions) Endpoint {
	c.checkInit()
	checkRequired(opts.Name, "Endpoint.Name MUST be specified")
	checkRequired(opts.RpcAddr, "Endpoint.RpcAddr MUST be specified")
	checkRequired(opts.InspectFunc, "Endpoint.InspectFunc MUST be specified")
	c.serviceType = "Endpoint"
	c.serviceName = checkNameFormat(opts.Name)
	return &implEndpoint{
		scoped:       c.scoped,
		name:         opts.Name,
		endpointAddr: opts.RpcAddr,
		inspectFunc:  opts.InspectFunc,
	}
}

func (c *implContext) NewDriver(opts DriverOptions) Driver {
	c.checkInit()
	checkRequired(opts.Name, "Driver.Name MUST be specified")
	checkRequired(opts.Topics, "Driver.Topics MUST be specified")
	c.serviceType = "Driver"
	c.serviceName = checkNameFormat(opts.Name)
	return &implDriver{
		scoped: c.scoped,
		name:   opts.Name,
		topics: opts.Topics,
	}
}

func (c *implContext) TermChan() <-chan os.Signal {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	signal.Ignore(syscall.SIGPIPE)
	return sig
}

func (c *implContext) TermAwait() error {
	<-c.TermChan()
	return nil
}

func (c *implContext) Log() *zap.SugaredLogger {
	return log
}

func (c *implContext) LogIfVerbose(fn func(log *zap.SugaredLogger)) {
	if c.logVerbose {
		fn(log)
	}
}

func newContext(global *GlobalScoped) Context {
	return &implContext{
		scoped:     global,
		logVerbose: "true" == os.Getenv(EnvKeyLogVerbose),
	}
}

func (c *implContext) checkInit() {
	if c.serviceType != "" {
		log.Panicf("Context已作为[%]服务使用", c.serviceType)
	}
}
