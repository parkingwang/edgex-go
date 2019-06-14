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
	NodeName

	// SendEventMessage 发送事件消息。指定 virtualDeviceName 的数据体。其中，
	// virtualDeviceName 为触发器内部的虚拟设备名称，它与与Trigger的节点名称组成完整的设备名称来作为消息来源。
	// 最终发送消息的Name字段，与Inspect返回的虚拟设备Name字段是相同的。
	SendEventMessage(virtualDevName string, data []byte) error
}

type TriggerOptions struct {
	NodeName    string         // 节点名称
	Topic       string         // 触发器发送事件的主题
	InspectFunc func() Inspect // Inspect消息生成函数
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

func (t *implTrigger) NodeName() string {
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

func (t *implTrigger) SendEventMessage(virtualName string, data []byte) error {
	// 构建完整的设备名称
	fullDeviceName := CreateVirtualDeviceName(t.nodeName, virtualName)
	token := t.mqttClient.Publish(
		t.mqttTopic,
		t.scoped.MqttQoS,
		t.scoped.MqttRetained,
		NewMessage([]byte(fullDeviceName), data).getFrames())
	if token.Wait() && nil != token.Error() {
		return errors.WithMessage(token.Error(), "发送事件消息出错")
	} else {
		return nil
	}
}

func (t *implTrigger) Shutdown() {
	t.shutdownCancel()
	t.mqttClient.Disconnect(t.scoped.MqttQuitMillSec)
}
