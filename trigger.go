package edgex

import (
	"context"
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
	// 返回命名
	Name() string
	// 发送事件消息
	SendEventMessage(b Message) error
	// 发送Alive消息
	SendAliveMessage(m Message) error
}

type TriggerOptions struct {
	Name        string
	Topic       string
	InspectFunc func() Inspect
}

//// trigger

type implTrigger struct {
	Trigger
	scoped   *GlobalScoped
	topic    string // Trigger产生的事件Topic
	nodeName string // Trigger的名称
	// Inspect
	inspectFunc func() Inspect
	// MQTT
	mqttClient mqtt.Client
	mqttTopic  string
	// Shutdown
	shutdownContext context.Context
	shutdownCancel  context.CancelFunc
}

func (t *implTrigger) Name() string {
	return t.nodeName
}

func (t *implTrigger) Startup() {
	t.shutdownContext, t.shutdownCancel = context.WithCancel(context.Background())
	// 连接Broker
	opts := mqtt.NewClientOptions()
	opts.SetClientID(fmt.Sprintf("EDGEX-TRIGGER:%s", t.nodeName))
	opts.SetWill(topicOfOffline("Trigger", t.nodeName), "offline", 1, true)
	mqttSetOptions(opts, t.scoped)
	t.mqttTopic = topicOfTrigger(t.topic)
	t.mqttClient = mqtt.NewClient(opts)
	log.Info("Mqtt客户端连接Broker: ", t.scoped.MqttBroker)
	// 连续重试
	mqttAwaitConnection(t.mqttClient, t.scoped.MqttMaxRetry)

	if !t.mqttClient.IsConnected() {
		log.Panic("Mqtt客户端连接无法连接Broker")
	} else {
		// 异步发送Inspect消息
		mqttSendInspectMessage(t.mqttClient, t.nodeName, t.inspectFunc)

		go mqttAsyncTickInspect(t.shutdownContext, func() {
			mqttSendInspectMessage(t.mqttClient, t.nodeName, t.inspectFunc)
		})
	}
}

func (t *implTrigger) SendEventMessage(b Message) error {
	token := t.mqttClient.Publish(
		t.mqttTopic,
		t.scoped.MqttQoS,
		t.scoped.MqttRetained,
		b.getFrames())
	if token.Wait() && nil != token.Error() {
		return errors.WithMessage(token.Error(), "发送事件消息出错")
	} else {
		return nil
	}
}

func (t *implTrigger) SendAliveMessage(alive Message) error {
	return mqttSendAliveMessage(t.mqttClient, "Trigger", t.nodeName, alive)
}

func (t *implTrigger) Shutdown() {
	t.shutdownCancel()
	t.mqttClient.Disconnect(t.scoped.MqttQuitMillSec)
}
