package edgex

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
	"time"
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
	EnvKeyMQBroker       = "EDGEX_MQTT_BROKER"
	EnvKeyMQUsername     = "EDGEX_MQTT_USERNAME"
	EnvKeyMQPassword     = "EDGEX_MQTT_PASSWORD"
	EnvKeyMQQOS          = "EDGEX_MQTT_QOS"
	EnvKeyMQRetained     = "EDGEX_MQTT_RETAINED"
	EnvKeyMQCleanSession = "EDGEX_MQTT_CLEAN_SESSION"
	EnvKeyConfig         = "EDGEX_CONFIG"
	EnvKeyLogVerbose     = "EDGEX_LOG_VERBOSE"
	EnvKeyGrpcAddress    = "EDGEX_GRPC_ADDRESS"

	MqttBrokerDefault = "tcp://mqtt-broker.edgex.io:1883"

	DefaultConfName = "application.toml"
	DefaultConfFile = "/etc/edgex/application.toml"
)

var (
	ErrConfigNotExist = errors.New("config not exists")
)

////

// Run 运行EdgeX节点服务
func Run(application func(ctx Context) error) {
	ctx := CreateDefaultContext()
	log.Debug("启动EdgeX-App")
	defer log.Debug("停止EdgeX-App")
	if err := application(ctx); nil != err {
		log.Error("EdgeX-App出错: ", err)
	}
}

// CreateContext 使用指定Scoped参数，创建Context对象。
func CreateContext(scoped *GlobalScoped) Context {
	return newContext(scoped)
}

// CreateDefaultContext 从环境变量中读取Scoped参数，并创建返回Context对象。
func CreateDefaultContext() Context {
	return CreateContext(&GlobalScoped{
		MqttBroker:            EnvGetString(EnvKeyMQBroker, MqttBrokerDefault),
		MqttUsername:          EnvGetString(EnvKeyMQUsername, ""),
		MqttPassword:          EnvGetString(EnvKeyMQPassword, ""),
		MqttQoS:               uint8(EnvGetInt64(EnvKeyMQQOS, 2)),
		MqttRetained:          EnvGetBoolean(EnvKeyMQRetained, false),
		MqttCleanSession:      EnvGetBoolean(EnvKeyMQCleanSession, true),
		MqttKeepAlive:         time.Second * 3,
		MqttPingTimeout:       time.Second * 1,
		MqttConnectTimeout:    time.Second * 5,
		MqttReconnectInterval: time.Second * 1,
		MqttAutoReconnect:     true,
		MqttMaxRetry:          120,
		MqttQuitMillSec:       500,
	})
}

//// Context实现

type implContext struct {
	scoped      *GlobalScoped
	serviceType string
	serviceName string
	logVerbose  bool
}

func (c *implContext) LoadConfig() map[string]interface{} {
	return LoadConfig()
}

func (c *implContext) NewTrigger(opts TriggerOptions) Trigger {
	c.checkInit()
	checkRequired(opts.NodeName, "Trigger.NodeName MUST be specified")
	checkRequired(opts.Topic, "Trigger.Topic MUST be specified")
	checkRequired(opts.InspectFunc, "Trigger.InspectFunc MUST be specified")
	c.serviceType = "Trigger"
	c.serviceName = checkNameFormat(opts.NodeName)
	return &implTrigger{
		scoped:      c.scoped,
		topic:       opts.Topic,
		nodeName:    opts.NodeName,
		inspectFunc: opts.InspectFunc,
	}
}

func (c *implContext) NewEndpoint(opts EndpointOptions) Endpoint {
	c.checkInit()
	checkRequired(opts.NodeName, "Endpoint.NodeName MUST be specified")
	checkRequired(opts.RpcAddr, "Endpoint.RpcAddr MUST be specified")
	checkRequired(opts.InspectFunc, "Endpoint.InspectFunc MUST be specified")
	c.serviceType = "Endpoint"
	c.serviceName = checkNameFormat(opts.NodeName)
	return &implEndpoint{
		scoped:       c.scoped,
		name:         opts.NodeName,
		endpointAddr: opts.RpcAddr,
		inspectFunc:  opts.InspectFunc,
	}
}

func (c *implContext) NewDriver(opts DriverOptions) Driver {
	c.checkInit()
	checkRequired(opts.NodeName, "Driver.NodeName MUST be specified")
	checkRequired(opts.Topics, "Driver.Topics MUST be specified")
	c.serviceType = "Driver"
	c.serviceName = checkNameFormat(opts.NodeName)
	return &implDriver{
		scoped: c.scoped,
		name:   opts.NodeName,
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

////

// CreateVirtualDeviceName 创建虚拟设备的完整名称
func CreateVirtualDeviceName(nodeName, virtualDeviceName string) string {
	return nodeName + ":" + virtualDeviceName
}

// LoadConfig 加载TOML配置文件
func LoadConfig() map[string]interface{} {
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
