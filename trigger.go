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

	// SendEventMessage 发送事件消息。指定 virtualNodeName 的数据体。其中，
	// virtualNodeName 为触发器内部的虚拟设备名称，它与Trigger的节点名称组成完整的设备名称来作为消息来源。
	// 最终发送消息的Name字段，与Inspect返回的虚拟设备Name字段是相同的。
	SendEventMessage(virtualNodeName string, data []byte) error
}

type TriggerOptions struct {
	NodeName    string         // 节点名称
	Topic       string         // 触发器发送事件的主题
	InspectFunc func() Inspect // Inspect消息生成函数
}

//// trigger

type NodeTrigger struct {
	Trigger
	globals  *Globals
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

func (t *NodeTrigger) NodeName() string {
	return t.nodeName
}

func (t *NodeTrigger) Startup() {
	t.shutdownContext, t.shutdownCancel = context.WithCancel(context.Background())
	// 连接Broker
	opts := mqtt.NewClientOptions()
	opts.SetClientID(fmt.Sprintf("EX-Trigger:%s", t.nodeName))
	opts.SetWill(topicOfOffline("Trigger", t.nodeName),
		"offline", 1, true)
	mqttSetOptions(opts, t.globals)
	t.mqttTopic = topicOfTrigger(t.topic)
	t.mqttClient = mqtt.NewClient(opts)
	log.Info("Mqtt客户端连接Broker: ", t.globals.MqttBroker)

	// 连续重试
	mqttAwaitConnection(t.mqttClient, t.globals.MqttMaxRetry)

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

func (t *NodeTrigger) SendEventMessage(virtualNodeName string, data []byte) error {
	// 构建完整的设备名称
	fullDeviceName := CreateVirtualDeviceName(t.nodeName, virtualNodeName)
	token := t.mqttClient.Publish(
		t.mqttTopic,
		t.globals.MqttQoS,
		t.globals.MqttRetained,
		NewMessage([]byte(fullDeviceName), data).getFrames())
	if token.Wait() && nil != token.Error() {
		return errors.WithMessage(token.Error(), "发送事件消息出错")
	} else {
		return nil
	}
}

func (t *NodeTrigger) Shutdown() {
	t.shutdownCancel()
	t.mqttClient.Disconnect(t.globals.MqttQuitMillSec)
}
