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

	// 当系统环境变量中，设置了"verbose"且为"true"时触发冗余日志输出操作。
	LogIfVerbose(fn func(log *zap.SugaredLogger))

	// LoadConfig 加载配置文件，返回Map数据结构对象。
	// 如果配置文件不存在，返回空Map数据结构，而非nil引用。
	LoadConfig() map[string]interface{}

	// NewTrigger 创建Trigger对象，并绑定Context为Trigger节点。
	NewTrigger(opts TriggerOptions) Trigger

	// NewEndpoint 创建Endpoint对象，并绑定Context为Endpoint节点。
	NewEndpoint(opts EndpointOptions) Endpoint

	// NewDriver 创建Driver对象，并绑定Context为Driver节点。
	NewDriver(opts DriverOptions) Driver

	// TermChan 返回监听系统中断退出信号的通道
	TermChan() <-chan os.Signal

	// TermAwait 阻塞等待系统中断退出信号
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
func CreateContext(scoped *Globals) Context {
	return newContext(scoped)
}

// CreateDefaultContext 从环境变量中读取Scoped参数，并创建返回Context对象。
func CreateDefaultContext() Context {
	return CreateContext(&Globals{
		MqttBroker:            EnvGetString(EnvKeyMQBroker, MqttBrokerDefault),
		MqttUsername:          EnvGetString(EnvKeyMQUsername, ""),
		MqttPassword:          EnvGetString(EnvKeyMQPassword, ""),
		MqttQoS:               uint8(EnvGetInt64(EnvKeyMQQOS, 1)),
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

type NodeContext struct {
	globals    *Globals
	logVerbose bool
}

func (c *NodeContext) LoadConfig() map[string]interface{} {
	return LoadConfig()
}

func (c *NodeContext) NewTrigger(opts TriggerOptions) Trigger {
	checkRequires(opts.NodeName, "Trigger.NodeName MUST be specified")
	checkRequires(opts.Topic, "Trigger.Topic MUST be specified")
	checkRequires(opts.AutoInspectFunc, "Trigger.AutoInspectFunc MUST be specified")
	return &trigger{
		globals:         c.globals,
		topic:           opts.Topic,
		nodeName:        opts.NodeName,
		sequenceId:      0,
		autoInspectFunc: opts.AutoInspectFunc,
	}
}

func (c *NodeContext) NewEndpoint(opts EndpointOptions) Endpoint {
	checkRequires(opts.NodeName, "Endpoint.NodeName MUST be specified")
	checkRequires(opts.RpcAddr, "Endpoint.RpcAddr MUST be specified")
	checkRequires(opts.AutoInspectFunc, "Endpoint.AutoInspectFunc MUST be specified")
	return &endpoint{
		globals:         c.globals,
		nodeName:        opts.NodeName,
		sequenceId:      0,
		endpointAddr:    opts.RpcAddr,
		autoInspectFunc: opts.AutoInspectFunc,
		serialExecuting: opts.SerialExecuting,
	}
}

func (c *NodeContext) NewDriver(opts DriverOptions) Driver {
	checkRequires(opts.NodeName, "Driver.NodeName MUST be specified")
	checkRequires(opts.Topics, "Driver.Topics MUST be specified")
	return &driver{
		globals:  c.globals,
		nodeName: opts.NodeName,
		topics:   opts.Topics,
	}
}

func (c *NodeContext) TermChan() <-chan os.Signal {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	signal.Ignore(syscall.SIGPIPE)
	return sig
}

func (c *NodeContext) TermAwait() error {
	<-c.TermChan()
	return nil
}

func (c *NodeContext) Log() *zap.SugaredLogger {
	return log
}

func (c *NodeContext) LogIfVerbose(fn func(log *zap.SugaredLogger)) {
	if c.logVerbose {
		fn(log)
	}
}

func newContext(global *Globals) Context {
	return &NodeContext{
		globals:    global,
		logVerbose: EnvGetBoolean(EnvKeyLogVerbose, false),
	}
}

////

// LoadConfig 加载TOML配置文件。
// 配置文件加载顺序：
// 1. 运行目录下的 application.toml;
// 2. /etc/edgex/application.toml;
// 3. 环境变量"EDGEX_CONFIG"指定的路径;
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
