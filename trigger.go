package edgex

import (
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// Trigger 触发器，用于产生事件
type Trigger interface {
	Lifecycle
	// 生产事件
	Triggered(b Packet) error
}

type TriggerOptions struct {
	Name  string
	Topic string
}

//// trigger 产生事件

type trigger struct {
	Trigger
	scoped *GlobalScoped
	topic  string // Trigger产生的事件Topic
	name   string // Trigger的名称
	// MQTT
	mqttClient mqtt.Client
	mqttTopic  string
}

func (t *trigger) Startup() {
	// 连接Broker
	opts := mqtt.NewClientOptions()
	opts.SetClientID(fmt.Sprintf("Trigger-%s", t.name))
	opts.SetWill(topicOfWill("Trigger", t.name), "offline", 1, true)
	setMqttDefaults(opts, t.scoped)

	t.mqttTopic = topicOfTrigger(t.topic)
	t.mqttClient = mqtt.NewClient(opts)
	log.Info("Mqtt客户端连接Broker: ", t.scoped.MqttBroker)
	if token := t.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Panic("Mqtt客户端连接出错：", token.Error())
	} else {
		log.Info("Mqtt客户端连接成功, TriggerTopic: ", t.mqttTopic)
	}
}

func (t *trigger) Triggered(b Packet) error {
	token := t.mqttClient.Publish(
		t.mqttTopic,
		t.scoped.MqttQoS,
		t.scoped.MqttRetained,
		b.Bytes())
	if token.Wait() && nil != token.Error() {
		return errors.WithMessage(token.Error(), "发送消息出错")
	} else {
		return nil
	}
}

func (t *trigger) Shutdown() {
	t.mqttClient.Disconnect(1000)
}
