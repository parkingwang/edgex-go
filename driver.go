package edgex

import (
	"context"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"math"
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

	// Execute 发起一个同步消息请求，并获取响应消息。如果过程中发生错误，返回错误消息。
	Execute(endpointAddr string, in Message, timeout time.Duration) (out Message, err error)

	// Hello 发起一个同步Hello消息，并获取响应消息。通常使用此函数来触发gRPC创建并预热两个节点之间的连接。
	// Hello 函数调用的是Execute函数，发送Ping消息，返回消息为Pong消息。
	Hello(endpointAddr string, timeout time.Duration) error

	// NextSequenceId 返回流水号
	NextSequenceId() uint32

	// 基于内部流水号创建消息对象
	NextMessage(virtualNodeId string, body []byte) Message
}

type DriverOptions struct {
	Topics []string // 监听主题列表
}

//// Driver实现

type driver struct {
	Driver
	globals    *Globals
	nodeName   string
	sequenceId uint32
	// MQTT
	topics           []string
	refMqttClient    mqtt.Client
	mqttTopicTrigger map[string]byte
	mqttWorker       func(event Message)
}

func (d *driver) NodeName() string {
	return d.nodeName
}

func (d *driver) NextSequenceId() uint32 {
	d.sequenceId = (d.sequenceId + 1) % math.MaxUint32
	return d.sequenceId
}

func (d *driver) NextMessage(virtualNodeId string, body []byte) Message {
	return NewMessageWithId(d.nodeName, virtualNodeId, body, d.NextSequenceId())
}

func (d *driver) Startup() {
	// 监听所有Trigger的UserTopic
	d.mqttTopicTrigger = make(map[string]byte, len(d.topics))
	for _, t := range d.topics {
		topic := topicOfEvents(t)
		d.mqttTopicTrigger[topic] = 0
		log.Info("开启监听事件[TRIGGER]: ", topic)
	}
	d.refMqttClient.SubscribeMultiple(d.mqttTopicTrigger, func(cli mqtt.Client, msg mqtt.Message) {
		frame := msg.Payload()
		if ok, err := CheckMessage(frame); !ok {
			log.Error("消息格式异常: ", err)
		} else {
			d.mqttWorker(ParseMessage(frame))
		}
	})
}

func (d *driver) Shutdown() {
	topics := make([]string, 0)
	for t := range d.mqttTopicTrigger {
		topics = append(topics, t)
		log.Info("取消监听事件[TRIGGER]: ", t)
	}
	if token := d.refMqttClient.Unsubscribe(topics...); token.Wait() && nil != token.Error() {
		log.Error("取消监听事件出错：", token.Error())
	}
}

func (d *driver) Process(f func(Message)) {
	d.mqttWorker = f
}

func (d *driver) Execute(endpointAddr string, in Message, to time.Duration) (out Message, err error) {
	log.Debug("GRPC调用Endpoint: ", endpointAddr)
	remote, err := grpc.Dial(endpointAddr, grpc.WithInsecure())
	if nil != err {
		return nil, errors.WithMessage(err, "gRPC_DIAL_ERROR")
	}
	client := NewExecuteClient(remote)
	ctx, cancel := context.WithTimeout(context.Background(), to)
	defer cancel()

	ret, err := client.Execute(ctx, &Data{Frames: in.Bytes()})

	if nil != err {
		return nil, err
	} else {
		return ParseMessage(ret.GetFrames()), nil
	}
}

func (d *driver) Hello(endpointAddr string, timeout time.Duration) error {
	_, err := d.Execute(
		endpointAddr,
		newControlMessageWithId(d.nodeName, d.nodeName, FrameVarPing, d.NextSequenceId()),
		timeout)
	return err
}
