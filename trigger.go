package edgex

import (
	"context"
	"github.com/beinan/fastid"
	"github.com/eclipse/paho.mqtt.golang"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// Trigger 触发器，用于产生事件。
type Trigger interface {
	NeedLifecycle
	NeedAccessNodeId
	NeedCreateMessages
	NeedInspectProperties

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
	Topic              string                    // 触发器发送事件的主题
	NodePropertiesFunc func() MainNodeProperties // Inspect消息生成函数
}

//// trigger

type trigger struct {
	Trigger
	nodeId   string // Trigger的名称
	opts     TriggerOptions
	globals  *Globals
	seqIdRef *fastid.Config // Trigger产生的消息ID序列
	// MQTT
	mqttRef        mqtt.Client
	mqttEventTopic string // MQTT使用的EventTopic
	mqttValueTopic string // MQTT使用的ValueTopic

	// Shutdown
	stopContext context.Context
	stopCancel  context.CancelFunc
}

func (t *trigger) NodeId() string {
	return t.nodeId
}

func (t *trigger) NextMessageSequenceId() int64 {
	return t.seqIdRef.GenInt64ID()
}

func (t *trigger) NextMessageBy(virtualId string, body []byte) Message {
	return NewMessageWith(t.nodeId, virtualId, body, t.NextMessageSequenceId())
}

func (t *trigger) NextMessageOf(virtualNodeId string, body []byte) Message {
	return NewMessageById(virtualNodeId, body, t.NextMessageSequenceId())
}

func (t *trigger) Startup() {
	t.stopContext, t.stopCancel = context.WithCancel(context.Background())
	// 重建Topic前缀
	t.mqttEventTopic = TopicOfEvents(t.opts.Topic)
	t.mqttValueTopic = TopicOfValues(t.opts.Topic)
	// 定时发送Properties消息
	if nil != t.opts.NodePropertiesFunc {
		prop := t.opts.NodePropertiesFunc()
		go scheduleSendProperties(t.stopContext, func() {
			t.PublishNodeProperties(prop)
		})
	}
}

func (t *trigger) PublishNodeProperties(properties MainNodeProperties) {
	t.checkReady()
	properties.NodeId = t.nodeId
	mqttSendNodeProperties(t.globals, t.mqttRef, properties)
}

func (t *trigger) PublishNodeState(state VirtualNodeState) {
	t.checkReady()
	state.NodeId = t.nodeId
	mqttSendNodeState(t.mqttRef, state)
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
	t.checkReady()
	token := t.mqttRef.Publish(
		mqttTopic,
		qos,
		retained,
		message.Bytes())
	if token.Wait() && nil != token.Error() {
		return token.Error()
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
	t.stopCancel()
}

func (t *trigger) checkReady() {
	if t.stopCancel == nil || t.stopContext == nil {
		log.Panic("Trigger未启动，须调用Startup()/Shutdown()")
	}
}
