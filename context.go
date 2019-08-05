package edgex

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/yoojia/go-value"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// Context 是一个提供基础通讯环境和参数设置的对象。通过Context来创建Trigger, Endpoint, Driver组件，并为组件提供MQTT通讯能力。
type Context interface {
	NeedNodeId
	// 使用默认配置结构来初化Context
	InitialWithConfig(config map[string]interface{})

	// 初化和设置Context
	Initial(nodeId string)

	// Destroy 由组件自动调用
	destroy()

	// 返回Log对象
	Log() *zap.SugaredLogger

	// 当系统环境变量中，设置了"verbose"且为"true"时触发冗余日志输出操作。
	LogIfVerbose(fn func(log *zap.SugaredLogger))

	// LoadConfig 加载默认配置文件名的配置
	LoadConfig() map[string]interface{}

	// LoadConfigByName 加载指定文件名的配置，返回Map数据结构对象。
	// 如果配置文件不存在，返回空Map数据结构，而非nil引用。
	LoadConfigByName(fileName string) map[string]interface{}

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
	DefaultConfDir  = "/etc/edgex/"
)

var (
	ErrConfigNotExist = errors.New("config not exists")
)

////

// Run 运行EdgeX节点服务
func Run(application func(ctx Context) error) {
	ctx := CreateDefaultContext()
	log.Info("启动EdgeX-App")
	defer func() {
		log.Info("停止EdgeX-App")
		ctx.destroy()
	}()
	if err := application(ctx); nil != err {
		log.Error("EdgeX-App出错: ", err)
	}
}

// CreateContext 使用指定 Globals 参数，创建Context对象。
func CreateContext(globals *Globals) Context {
	return newContext(globals)
}

// CreateDefaultContext 从环境变量中读取 Globals 参数，并创建返回Context对象。
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
	nodeId     string
	mqttClient mqtt.Client
}

func (c *NodeContext) InitialWithConfig(config map[string]interface{}) {
	c.nodeId = value.ToString(config["NodeId"])
	checkIdFormat("NodeId", c.nodeId)
	// Globals设置
	if globals, ok := value.ToMap(config["Globals"]); ok {
		// 其它全局配置
		if flag, ok := value.ToBool(globals["LogVerbose"]); ok {
			c.logVerbose = flag
		}
		// MQTT配置
		if str, ok := value.ToStringB(globals["MqttBroker"]); ok {
			c.globals.MqttBroker = str
		}
		if str, ok := value.ToStringB(globals["MqttUsername"]); ok {
			c.globals.MqttUsername = str
		}
		if str, ok := value.ToStringB(globals["MqttPassword"]); ok {
			c.globals.MqttPassword = str
		}
		if iv, ok := value.ToInt64(globals["MqttQoS"]); ok {
			c.globals.MqttQoS = uint8(iv)
		}
		if flag, ok := value.ToBool(globals["MqttRetained"]); ok {
			c.globals.MqttRetained = flag
		}
		if du, ok := value.ToDuration(globals["MqttKeepAlive"]); ok {
			c.globals.MqttKeepAlive = du
		}
		if du, ok := value.ToDuration(globals["MqttPingTimeout"]); ok {
			c.globals.MqttPingTimeout = du
		}
		if du, ok := value.ToDuration(globals["MqttConnectTimeout"]); ok {
			c.globals.MqttConnectTimeout = du
		}
		if du, ok := value.ToDuration(globals["MqttReconnectInterval"]); ok {
			c.globals.MqttReconnectInterval = du
		}
		if flag, ok := value.ToBool(globals["MqttAutoReconnect"]); ok {
			c.globals.MqttAutoReconnect = flag
		}
		if flag, ok := value.ToBool(globals["MqttCleanSession"]); ok {
			c.globals.MqttCleanSession = flag
		}
		if iv, ok := value.ToInt64(globals["MqttMaxRetry"]); ok {
			c.globals.MqttMaxRetry = int(iv)
		}
		if iv, ok := value.ToInt64(globals["MqttQuitMillSec"]); ok {
			c.globals.MqttQuitMillSec = uint(iv)
		}
	}
	// MQTT Broker
	opts := mqtt.NewClientOptions()
	clientId := fmt.Sprintf("%s:%s", MqttClientIdHeader, c.nodeId)
	opts.SetClientID(clientId)
	opts.SetWill(topicOfOffline(MqttClientIdHeader, c.nodeId), "offline", 1, true)
	mqttSetOptions(opts, c.globals)
	c.mqttClient = mqtt.NewClient(opts)
	log.Info("Mqtt客户端连接Broker: ", c.globals.MqttBroker)

	// 连续重试
	mqttAwaitConnection(c.mqttClient, c.globals.MqttMaxRetry)

	if !c.mqttClient.IsConnected() {
		log.Panic("Mqtt客户端连接无法连接Broker")
	} else {
		log.Info("Mqtt客户端连接成功：" + clientId)
	}
}

func (c *NodeContext) Initial(nodeId string) {
	c.InitialWithConfig(map[string]interface{}{
		"NodeId": nodeId,
	})
}

func (c *NodeContext) NodeId() string {
	return c.nodeId
}

func (c *NodeContext) destroy() {
	c.mqttClient.Disconnect(c.globals.MqttQuitMillSec)
}

func (c *NodeContext) LoadConfig() map[string]interface{} {
	return LoadConfig()
}

func (c *NodeContext) LoadConfigByName(fileName string) map[string]interface{} {
	return LoadConfigByName(fileName)
}

func (c *NodeContext) NewTrigger(opts TriggerOptions) Trigger {
	c.checkInit()
	checkRequires(opts.Topic, "Trigger.Topic MUST be specified")
	checkRequires(opts.AutoInspectFunc, "Trigger.AutoInspectFunc MUST be specified")
	return &trigger{
		mqttRef:         c.mqttClient,
		globals:         c.globals,
		topic:           opts.Topic,
		nodeId:          c.nodeId,
		sequenceId:      0,
		autoInspectFunc: opts.AutoInspectFunc,
	}
}

func (c *NodeContext) NewEndpoint(opts EndpointOptions) Endpoint {
	c.checkInit()
	checkRequires(opts.AutoInspectFunc, "Endpoint.AutoInspectFunc MUST be specified")
	return &endpoint{
		mqttRef:         c.mqttClient,
		globals:         c.globals,
		nodeId:          c.nodeId,
		sequenceId:      0,
		autoInspectFunc: opts.AutoInspectFunc,
	}
}

func (c *NodeContext) NewDriver(opts DriverOptions) Driver {
	c.checkInit()
	checkRequires(opts.EventTopics, "Driver.Topics MUST be specified")
	return &driver{
		mqttRef: c.mqttClient,
		globals: c.globals,
		nodeId:  c.nodeId,
		opts:    opts,
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

func (c *NodeContext) checkInit() {
	if nil == c.mqttClient {
		log.Panic("Context未初始化")
	}
}

func newContext(global *Globals) Context {
	return &NodeContext{
		globals:    global,
		logVerbose: EnvGetBoolean(EnvKeyLogVerbose, false),
	}
}

////

// LoadConfigByName 加载指定文件名的配置信息。
// 配置文件加载顺序：
// 1. 当前运行目录;
// 2. 目录：/etc/edgex/;
// 3. 环境变量"EDGEX_CONFIG"指定的路径;
func LoadConfigByName(fileName string) map[string]interface{} {
	searchConfig := func(files ...string) (f string, err error) {
		for _, file := range files {
			if _, err := os.Stat(file); nil == err {
				return file, nil
			}
		}
		return "", ErrConfigNotExist
	}
	config := make(map[string]interface{})
	file, err := searchConfig(fileName, DefaultConfDir+fileName, os.Getenv(EnvKeyConfig))
	if nil != err {
		log.Panic("未设置任何配置文件")
	} else {
		log.Info("加载配置文件：", file)
	}
	if _, err := toml.DecodeFile(file, &config); nil != err {
		log.Error(fmt.Sprintf("读取配置文件(%s)出错: ", file), err)
	}
	return config
}

// LoadConfig 加载默认文件名的配置。
func LoadConfig() map[string]interface{} {
	return LoadConfigByName(DefaultConfName)
}
