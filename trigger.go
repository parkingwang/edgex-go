package edgex

import (
	"context"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"math"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// Trigger 触发器，用于产生事件
type Trigger interface {
	Lifecycle
	NodeName

	// PublishEvent 发送Event消息。发送消息的QoS使用默认设置。
	PublishEvent(virtualNodeId string, data []byte) error

	// SendEvent 发送精确的Event消息。使用QoS 2质量。
	SendEvent(virtualNodeId string, data []byte) error

	// PublishValue 发送Value数据消息。发送消息的QoS使用默认设置。
	PublishValue(virtualNodeId string, data []byte) error

	// SendValue 发送精确的Value消息。使用QoS 2质量。
	SendValue(virtualNodeId string, data []byte) error

	// 发布Inspect消息
	PublishInspect(node MainNode)

	// NextSequenceId 返回流水号
	NextSequenceId() uint32

	// 基于内部流水号创建消息对象
	NextMessage(virtualNodeId string, body []byte) Message
}

type TriggerOptions struct {
	Topic           string          // 触发器发送事件的主题
	AutoInspectFunc func() MainNode // Inspect消息生成函数
}

//// trigger

type trigger struct {
	Trigger
	globals        *Globals
	topic          string // Trigger的Topic
	mqttEventTopic string // MQTT使用的EventTopic
	mqttValueTopic string // MQTT使用的ValueTopic
	nodeName       string // Trigger的名称
	sequenceId     uint32 // Trigger产生的消息ID序列
	// MainNode 消息生产函数
	autoInspectFunc func() MainNode
	// MQTT
	refMqttClient mqtt.Client
	// Shutdown
	shutdownContext context.Context
	shutdownCancel  context.CancelFunc
}

func (t *trigger) NodeName() string {
	return t.nodeName
}

func (t *trigger) NextSequenceId() uint32 {
	t.sequenceId = (t.sequenceId + 1) % math.MaxUint32
	return t.sequenceId
}

func (t *trigger) NextMessage(virtualNodeId string, body []byte) Message {
	return NewMessageWithId(t.nodeName, virtualNodeId, body, t.NextSequenceId())
}

func (t *trigger) Startup() {
	t.shutdownContext, t.shutdownCancel = context.WithCancel(context.Background())
	// 重建Topic前缀
	t.mqttEventTopic = topicOfEvents(t.topic)
	t.mqttValueTopic = topicOfValues(t.topic)
	// 定时发送Inspect消息
	if nil != t.autoInspectFunc {
		go scheduleSendInspect(t.shutdownContext, func() {
			t.PublishInspect(t.autoInspectFunc())
		})
	}
}

func (t *trigger) PublishInspect(node MainNode) {
	mqttSendInspectMessage(t.refMqttClient, t.nodeName, node)
}

func (t *trigger) PublishEvent(virtualNodeId string, data []byte) error {
	return t.publish(t.mqttEventTopic, virtualNodeId, data, t.globals.MqttQoS, t.globals.MqttRetained)
}

func (t *trigger) SendEvent(virtualNodeId string, data []byte) error {
	return t.publish(t.mqttEventTopic, virtualNodeId, data, 2, true)
}

func (t *trigger) PublishValue(virtualNodeId string, data []byte) error {
	return t.publish(t.mqttValueTopic, virtualNodeId, data, t.globals.MqttQoS, t.globals.MqttRetained)
}

func (t *trigger) SendValue(virtualNodeId string, data []byte) error {
	return t.publish(t.mqttValueTopic, virtualNodeId, data, 2, true)
}

func (t *trigger) publish(topic string, virtualNodeId string, data []byte, qos byte, retained bool) error {
	// 构建完整的设备名称
	token := t.refMqttClient.Publish(
		topic,
		qos,
		retained,
		NewMessageWithId(t.nodeName, virtualNodeId, data, t.NextSequenceId()).Bytes())
	if token.Wait() && nil != token.Error() {
		return errors.WithMessage(token.Error(), "发送事件消息出错")
	} else {
		return nil
	}
}

func (t *trigger) Shutdown() {
	t.shutdownCancel()
}
