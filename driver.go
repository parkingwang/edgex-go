package edgex

import (
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
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

var ErrExecuteTimeout = errors.New("execute timeout")

//// Endpoint实现

type driver struct {
	Endpoint
	scoped  *GlobalScoped
	name    string
	replies map[string]chan Packet
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
	opts.AddBroker(d.scoped.MqttBroker)
	opts.SetKeepAlive(d.scoped.MqttKeepAlive)
	opts.SetPingTimeout(d.scoped.MqttPingTimeout)
	opts.SetAutoReconnect(d.scoped.MqttAutoReconnect)
	opts.SetConnectTimeout(d.scoped.MqttConnectTimeout)
	opts.SetWill(topicOfWill("Driver", d.name), "offline", 1, true)
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
		log.Info("监听Trigger.EventTopic: ", topic)
	}
	d.mqttClient.SubscribeMultiple(d.mqttTopicTrigger, func(cli mqtt.Client, msg mqtt.Message) {
		d.mqttWorker(PacketOfBytes(msg.Payload()))
	})

	// 监听所有Endpoint的Reply
	d.mqttTopicEndpoint = topicOfEndpointReplyQ("+")
	log.Info("监听Endpoint.ReplyTopic: ", d.mqttTopicEndpoint)
	d.mqttClient.Subscribe(d.mqttTopicEndpoint, 0, func(cli mqtt.Client, msg mqtt.Message) {
		endpointId := endpointIdOfReplyQ(msg.Topic())
		if ch, ok := d.replies[endpointId]; ok {
			ch <- PacketOfBytes(msg.Payload())
		} else {
			log.Debug("Endpoint没有接收队列: ", endpointId)
		}
	})
}

func (d *driver) Shutdown() {
	d.mqttClient.Disconnect(1000)
}

func (d *driver) Process(f func(Packet)) {
	d.mqttWorker = f
}

func (d *driver) Execute(endpointId string, in Packet, to time.Duration) (out Packet, err error) {
	var ch chan Packet
	if c, ok := d.replies[endpointId]; !ok {
		ch = make(chan Packet, 1)
		d.replies[endpointId] = ch
	} else {
		ch = c
	}
	// Send and wait
	if err := d.sendRequest(endpointId, in); nil != err {
		return []byte{}, err
	}

	select {
	case <-time.After(to):
		return []byte{}, ErrExecuteTimeout

	case p := <-ch:
		return p, nil
	}
}

func (d *driver) sendRequest(endpointId string, data Packet) error {
	token := d.mqttClient.Publish(
		topicOfEndpointRequestQ(endpointId),
		d.scoped.MqttQoS,
		d.scoped.MqttRetained, data.Bytes())
	if token.Wait() && nil != token.Error() {
		return errors.WithMessage(token.Error(), "发送消息出错")
	} else {
		return nil
	}
}
