package edgex

import (
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"github.com/yoojia/edgex/util"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

type Driver interface {
	Lifecycle
	// 处理消息
	Process(func(event Packet))
	// 发起一个消息请求，并获取响应消息。
	// 如果过程中发生错误，返回错误消息
	Execute(endpointId string, in Packet, timeout time.Duration) (out Packet, err error)
}

type DriverOptions struct {
	Name   string
	Topics []string
}

//// Endpoint实现

type driver struct {
	Endpoint
	scoped  *GlobalScoped
	name    string
	replies *util.MapX
	// MQTT
	topics            []string
	mqttClient        mqtt.Client
	mqttTopicTrigger  map[string]byte
	mqttTopicEndpoint string
	mqttWorker        func(event Packet)
}

func (d *driver) Startup() {
	// 连接Broker
	opts := mqtt.NewClientOptions()
	opts.SetClientID(fmt.Sprintf("Driver-%s", d.name))
	opts.SetWill(topicOfWill("Driver", d.name), "offline", 1, true)
	setMqttDefaults(opts, d.scoped)

	d.mqttClient = mqtt.NewClient(opts)
	log.Info("Mqtt客户端连接Broker: ", d.scoped.MqttBroker)
	if token := d.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Panic("Mqtt客户端连接出错：", token.Error())
	} else {
		log.Info("Mqtt客户端连接成功")
	}

	// 监听所有Trigger的UserTopic
	d.mqttTopicTrigger = make(map[string]byte, len(d.topics))
	for _, t := range d.topics {
		topic := topicOfTrigger(t)
		d.mqttTopicTrigger[topic] = 0
		log.Info("开启监听事件[TRIGGER]: ", topic)
	}
	d.mqttClient.SubscribeMultiple(d.mqttTopicTrigger, func(cli mqtt.Client, msg mqtt.Message) {
		d.mqttWorker(PacketOfBytes(msg.Payload()))
	})

	// 监听所有Endpoint的Reply
	d.mqttTopicEndpoint = topicOfEndpointReplyQ("+")
	log.Info("开启监听事件[ENDPOINT]: ", d.mqttTopicEndpoint)
	d.mqttClient.Subscribe(d.mqttTopicEndpoint, 0, func(cli mqtt.Client, msg mqtt.Message) {
		endpointId := endpointIdOfReplyQ(msg.Topic())
		log.Debug("接收到[ENDPOINT]响应事件：", endpointId)
		if v, ok := d.replies.Load(endpointId); ok {
			(v.(util.Lazy)).Store(PacketOfBytes(msg.Payload()))
		} else {
			log.Debug("Endpoint没有接收队列: ", endpointId)
		}
	})
}

func (d *driver) Shutdown() {
	topics := make([]string, 0)
	topics = append(topics, d.mqttTopicEndpoint)
	log.Info("取消监听事件[ENDPOINT]: ", d.mqttTopicEndpoint)
	for t := range d.mqttTopicTrigger {
		topics = append(topics, t)
		log.Info("取消监听事件[TRIGGER]: ", t)
	}
	if token := d.mqttClient.Unsubscribe(topics...); token.Wait() && nil != token.Error() {
		log.Error("取消监听事件出错：", token.Error())
	}
	d.mqttClient.Disconnect(1000)
}

func (d *driver) Process(f func(Packet)) {
	d.mqttWorker = f
}

func (d *driver) Execute(endpointId string, in Packet, to time.Duration) (out Packet, err error) {
	lazy, _ := d.replies.LoadOrStore(endpointId, func() interface{} {
		return util.NewLazyPacket()
	})
	// Send and wait
	if err := d.sendRequest(endpointId, in); nil != err {
		return []byte{}, err
	} else {
		if v, err := lazy.(util.Lazy).Take(to); nil != err {
			return nil, err
		} else {
			return v.(Packet), nil
		}
	}
}

func (d *driver) sendRequest(endpointId string, data Packet) error {
	topic := topicOfEndpointRequestQ(endpointId)
	log.Debug("发起Request请求：", topic)
	token := d.mqttClient.Publish(
		topic,
		d.scoped.MqttQoS,
		d.scoped.MqttRetained, data.Bytes())
	if token.Wait() && nil != token.Error() {
		return errors.WithMessage(token.Error(), "发送消息出错")
	} else {
		return nil
	}
}
