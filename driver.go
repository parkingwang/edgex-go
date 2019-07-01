package edgex

import (
	"context"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

type Driver interface {
	Lifecycle
	NodeName

	// Process 处理消息
	Process(func(event Message))

	// 发起一个消息请求，并获取响应消息。
	// 如果过程中发生错误，返回错误消息
	Execute(endpointAddr string, in Message, timeout time.Duration) (out Message, err error)
}

type DriverOptions struct {
	NodeName string   // 节点名称
	Topics   []string // 监听主题列表
}

//// Driver实现

type NodeDriver struct {
	Driver
	globals  *Globals
	nodeName string
	// MQTT
	topics           []string
	mqttClient       mqtt.Client
	mqttTopicTrigger map[string]byte
	mqttWorker       func(event Message)
}

func (d *NodeDriver) NodeName() string {
	return d.nodeName
}

func (d *NodeDriver) Startup() {
	// 连接Broker
	clientId := fmt.Sprintf("EX-Driver-%s", d.nodeName)
	opts := mqtt.NewClientOptions()
	opts.SetClientID(clientId)
	opts.SetWill(topicOfOffline("Driver", d.nodeName),
		"offline", 1, true)
	mqttSetOptions(opts, d.globals)

	d.mqttClient = mqtt.NewClient(opts)
	log.Info("Mqtt客户端连接Broker: ", d.globals.MqttBroker)

	// 连续重试
	mqttAwaitConnection(d.mqttClient, d.globals.MqttMaxRetry)

	if !d.mqttClient.IsConnected() {
		log.Panic("Mqtt客户端连接无法连接Broker")
	}else{
		log.Debug("Mqtt客户端连接成功: " + clientId)
	}

	// 监听所有Trigger的UserTopic
	d.mqttTopicTrigger = make(map[string]byte, len(d.topics))
	for _, t := range d.topics {
		topic := topicOfTrigger(t)
		d.mqttTopicTrigger[topic] = 0
		log.Info("开启监听事件[TRIGGER]: ", topic)
	}
	d.mqttClient.SubscribeMultiple(d.mqttTopicTrigger, func(cli mqtt.Client, msg mqtt.Message) {
		d.mqttWorker(ParseMessage(msg.Payload()))
	})
}

func (d *NodeDriver) Shutdown() {
	topics := make([]string, 0)
	for t := range d.mqttTopicTrigger {
		topics = append(topics, t)
		log.Info("取消监听事件[TRIGGER]: ", t)
	}
	if token := d.mqttClient.Unsubscribe(topics...); token.Wait() && nil != token.Error() {
		log.Error("取消监听事件出错：", token.Error())
	}
	d.mqttClient.Disconnect(d.globals.MqttQuitMillSec)
}

func (d *NodeDriver) Process(f func(Message)) {
	d.mqttWorker = f
}

func (d *NodeDriver) Execute(endpointAddr string, in Message, to time.Duration) (out Message, err error) {
	log.Debug("GRPC调用Endpoint: ", endpointAddr)
	remote, err := grpc.Dial(endpointAddr, grpc.WithInsecure())
	if nil != err {
		return nil, errors.WithMessage(err, "gRPC_DIAL_ERROR")
	}
	client := NewExecuteClient(remote)
	ctx, cancel := context.WithTimeout(context.Background(), to)
	defer cancel()

	ret, err := client.Execute(ctx, &Data{
		Frames: in.getFrames(),
	})

	if nil != err {
		return nil, err
	} else {
		return ParseMessage(ret.GetFrames()), nil
	}

}
