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

// Trigger 触发器，用于产生事件。
type Trigger interface {
	NeedLifecycle
	NeedNodeId
	NeedMessages
	NeedInspect

	// 发送MQTT消息
	PublishMqtt(mqttTopic string, message Message, qos uint8, retained bool) error

	// PublishEvent 发送虚拟节点的Event消息。发送消息的QoS使用默认设置。
	PublishEvent(virtualId string, data []byte) error

	// PublishEventWith 发送虚拟节点的Event消息。指定QOS参数。
	PublishEventWith(virtualId string, data []byte, qos uint8, retained bool) error

	// PublishValue 发送虚拟节点的Value消息。发送消息的QoS使用默认设置。
	PublishValue(virtualId string, data []byte) error

	// PublishValueWith 发送虚拟节点的Value消息。指定QOS参数。
	PublishValueWith(virtualId string, data []byte, qos uint8, retained bool) error
}

type TriggerOptions struct {
	Topic           string                    // 触发器发送事件的主题
	AutoInspectFunc func() MainNodeProperties // Inspect消息生成函数
}

//// trigger

type trigger struct {
	Trigger
	globals        *Globals
	topic          string // Trigger的Topic
	mqttEventTopic string // MQTT使用的EventTopic
	mqttValueTopic string // MQTT使用的ValueTopic
	nodeId         string // Trigger的名称
	sequenceId     uint32 // Trigger产生的消息ID序列
	// MainNodeProperties 消息生产函数
	autoInspectFunc func() MainNodeProperties
	// MQTT
	mqttRef mqtt.Client
	// Shutdown
	shutdownContext context.Context
	shutdownCancel  context.CancelFunc
}

func (t *trigger) NodeId() string {
	return t.nodeId
}

func (t *trigger) NextMessageSequenceId() uint32 {
	t.sequenceId = (t.sequenceId + 1) % math.MaxUint32
	return t.sequenceId
}

func (t *trigger) NextMessageBy(virtualId string, body []byte) Message {
	return NewMessageWith(t.nodeId, virtualId, body, t.NextMessageSequenceId())
}

func (t *trigger) NextMessageOf(virtualNodeId string, body []byte) Message {
	return NewMessageById(virtualNodeId, body, t.NextMessageSequenceId())
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

func (t *trigger) PublishInspect(node MainNodeProperties) {
	mqttSendInspectMessage(t.mqttRef, t.nodeId, node)
}

func (t *trigger) PublishEvent(virtualId string, data []byte) error {
	return t.PublishEventWith(virtualId, data, t.globals.MqttQoS, t.globals.MqttRetained)
}

func (t *trigger) PublishEventWith(virtualId string, data []byte, qos uint8, retained bool) error {
	return t.publishMessage(t.mqttEventTopic, virtualId, data, qos, retained)
}

func (t *trigger) PublishValue(virtualId string, data []byte) error {
	return t.PublishValueWith(virtualId, data, t.globals.MqttQoS, t.globals.MqttRetained)
}

func (t *trigger) PublishValueWith(virtualId string, data []byte, qos uint8, retained bool) error {
	return t.publishMessage(t.mqttValueTopic, virtualId, data, qos, retained)
}

func (t *trigger) PublishMqtt(mqttTopic string, message Message, qos uint8, retained bool) error {
	token := t.mqttRef.Publish(
		mqttTopic,
		qos,
		retained,
		message.Bytes())
	if token.Wait() && nil != token.Error() {
		return errors.WithMessage(token.Error(), "发送事件消息出错")
	} else {
		return nil
	}
}

func (t *trigger) publishMessage(mqttTopic string, virtualId string, data []byte, qos byte, retained bool) error {
	return t.PublishMqtt(mqttTopic,
		NewMessageWith(t.nodeId, virtualId, data, t.NextMessageSequenceId()),
		qos, retained)
}

func (t *trigger) Shutdown() {
	t.shutdownCancel()
}
