package edgex

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"strings"
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
	TermChan() <-chan os.Signal

	// 阻塞等待终止信号
	TermAwait() error
}

////

func Run(handler func(ctx Context) error) {
	broker, ok := os.LookupEnv("MQTTBroker")
	if !ok {
		broker = "tcp://mqtt-broker.edgex.io:1883"
	}
	scoped := NewDefaultGlobalScoped(broker)
	ctx := newContext(scoped)
	log.Debug("启动Service")
	defer log.Debug("停止Service")
	if err := handler(ctx); nil != err {
		log.Error("Service出错: ", err)
	}
}

const (
	AppConfEnvKey   = "EDGE_X_CONFIG"
	DefaultConfName = "application.toml"
	DefaultConfFile = "/etc/edgex/application.toml"
)

var (
	ErrConfigNotExist = errors.New("config not exists")
)

//// Context实现

type implContext struct {
	scoped      *GlobalScoped
	serviceType string
	serviceName string
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
	file, err := searchConfig(DefaultConfName, DefaultConfFile, os.Getenv(AppConfEnvKey))
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
	checkContextInitialize(c)
	checkRequired(opts.Name, "Trigger.Name MUST be specified")
	checkRequired(opts.Topic, "Trigger.Topic MUST be specified")
	checkRequired(opts.InspectFunc, "Trigger.InspectFunc MUST be specified")
	c.serviceType = "Trigger"
	c.serviceName = checkName(opts.Name)
	return &implTrigger{
		scoped:      c.scoped,
		topic:       opts.Topic,
		name:        opts.Name,
		inspectFunc: opts.InspectFunc,
	}
}

func (c *implContext) NewEndpoint(opts EndpointOptions) Endpoint {
	checkContextInitialize(c)
	checkRequired(opts.Name, "Endpoint.Name MUST be specified")
	checkRequired(opts.RpcAddr, "Endpoint.RpcAddr MUST be specified")
	checkRequired(opts.InspectFunc, "Endpoint.InspectFunc MUST be specified")
	c.serviceType = "Endpoint"
	c.serviceName = checkName(opts.Name)
	return &implEndpoint{
		scoped:       c.scoped,
		name:         opts.Name,
		endpointAddr: opts.RpcAddr,
		inspectFunc:  opts.InspectFunc,
	}
}

func (c *implContext) NewDriver(opts DriverOptions) Driver {
	checkContextInitialize(c)
	checkRequired(opts.Name, "Driver.Name MUST be specified")
	checkRequired(opts.Topics, "Driver.Topics MUST be specified")
	c.serviceType = "Driver"
	c.serviceName = checkName(opts.Name)
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

func newContext(global *GlobalScoped) Context {
	return &implContext{
		scoped: global,
	}
}

func checkName(name string) string {
	if "" == name || strings.Contains(name, "/") {
		log.Panic("服务名称中不能包含'/'字符")
	}
	return name
}

func checkContextInitialize(c *implContext) {
	if c.serviceType != "" {
		log.Panicf("Context已作为[%]服务使用", c.serviceType)
	}
}

func checkRequired(value interface{}, message string) {
	switch value.(type) {
	case string:
		if "" == value {
			log.Panic(message)
		}

	case []string:
		if 0 == len(value.([]string)) {
			log.Panic(message)
		}

	default:
		if nil == value {
			log.Panic(message)
		}
	}

}
