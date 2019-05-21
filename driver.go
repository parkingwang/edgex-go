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
	// 处理消息
	Process(func(event Message))
	// 发起一个消息请求，并获取响应消息。
	// 如果过程中发生错误，返回错误消息
	Execute(endpointAddr string, in Message, timeout time.Duration) (out Message, err error)
}

type DriverOptions struct {
	Name   string
	Topics []string
}

//// Driver实现

type implDriver struct {
	Driver
	scoped *GlobalScoped
	name   string
	// MQTT
	topics           []string
	mqttClient       mqtt.Client
	mqttTopicTrigger map[string]byte
	mqttWorker       func(event Message)
}

func (d *implDriver) Startup() {
	// 连接Broker
	opts := mqtt.NewClientOptions()
	opts.SetClientID(fmt.Sprintf("Driver-%s", d.name))
	opts.SetWill(topicOfWill("Driver", d.name), "offline", 1, true)
	setMqttDefaults(opts, d.scoped)

	d.mqttClient = mqtt.NewClient(opts)
	log.Info("Mqtt客户端连接Broker: ", d.scoped.MqttBroker)

	// 连续重试
	for retry := 0; retry < d.scoped.MqttMaxRetry; retry++ {
		if token := d.mqttClient.Connect(); token.Wait() && token.Error() != nil {
			log.Error("Mqtt客户端连接出错：", token.Error())
			<-time.After(time.Second)
		} else {
			log.Info("Mqtt客户端连接成功")
			break
		}
	}
	if !d.mqttClient.IsConnected() {
		log.Panic("Mqtt客户端连接无法连接Broker")
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

func (d *implDriver) Shutdown() {
	topics := make([]string, 0)
	for t := range d.mqttTopicTrigger {
		topics = append(topics, t)
		log.Info("取消监听事件[TRIGGER]: ", t)
	}
	if token := d.mqttClient.Unsubscribe(topics...); token.Wait() && nil != token.Error() {
		log.Error("取消监听事件出错：", token.Error())
	}
	d.mqttClient.Disconnect(1000)
}

func (d *implDriver) Process(f func(Message)) {
	d.mqttWorker = f
}

func (d *implDriver) Execute(endpointAddr string, in Message, to time.Duration) (out Message, err error) {
	log.Debug("GRPC调用Endpoint: ", endpointAddr)
	remote, err := grpc.Dial(endpointAddr, grpc.WithInsecure())
	if nil != err {
		return nil, errors.WithMessage(err, "GRPC DIAL ERROR")
	}
	client := NewExecuteClient(remote)
	ctx, cancel := context.WithTimeout(context.Background(), to)
	defer cancel()

	ret, err := client.Execute(ctx, &Data{
		Frames: in.getFrames(),
	})

	if nil != err && nil != ret {
		return ParseMessage(ret.GetFrames()), nil
	} else {
		return nil, err
	}
}
