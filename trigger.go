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

// Trigger 触发器，用于产生事件
type Trigger interface {
	Lifecycle
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
	scoped        *GlobalScoped
	topic         string // Trigger产生的事件Topic
	name          string // Trigger的名称
	inspectFunc   func() Inspect
	inspectTicker *time.Ticker
	// MQTT
	mqttClient mqtt.Client
	mqttTopic  string
}

func (t *implTrigger) Startup() {
	// 连接Broker
	opts := mqtt.NewClientOptions()
	opts.SetClientID(fmt.Sprintf("Trigger-%s", t.name))
	opts.SetWill(topicOfOffline("Trigger", t.name), "offline", 1, true)
	mqttSetOptions(opts, t.scoped)
	t.mqttTopic = topicOfTrigger(t.topic)
	t.mqttClient = mqtt.NewClient(opts)
	log.Info("Mqtt客户端连接Broker: ", t.scoped.MqttBroker)
	// 连续重试
	mqttRetryConnect(t.mqttClient, t.scoped.MqttMaxRetry)
	if !t.mqttClient.IsConnected() {
		log.Panic("Mqtt客户端连接无法连接Broker")
	} else {
		mqttSendInspectMessage(t.mqttClient, t.inspectFunc)
		go func() {
			t.inspectTicker = time.NewTicker(time.Minute)
			for range t.inspectTicker.C {
				mqttSendInspectMessage(t.mqttClient, t.inspectFunc)
			}
		}()
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
	return mqttSendAliveMessage(t.mqttClient, "Trigger", t.name, alive)
}

func (t *implTrigger) Shutdown() {
	t.inspectTicker.Stop()
	t.mqttClient.Disconnect(1000)
}
