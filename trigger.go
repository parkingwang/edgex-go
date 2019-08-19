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
	NeedProperties

	// 发送MQTT消息
	PublishMqtt(mqttTopic string, message Message, qos uint8, retained bool) error

	// PublishEvent 发送虚拟节点的Event消息。发送消息的QoS使用默认设置。
	PublishEvent(virtualId string, data []byte, eventId int64) error

	// PublishEventWith 发送虚拟节点的Event消息。指定QOS参数。
	PublishEventWith(virtualId string, data []byte, eventId int64, qos uint8, retained bool) error

	// PublishValue 发送虚拟节点的Value消息。发送消息的QoS使用默认设置。
	PublishValue(virtualId string, data []byte, eventId int64) error

	// PublishValueWith 发送虚拟节点的Value消息。指定QOS参数。
	PublishValueWith(virtualId string, data []byte, eventId int64, qos uint8, retained bool) error

	// PublishAction 发送虚拟节点的Action发送消息的QoS使用默认设置。
	PublishAction(virtualId string, data []byte, eventId int64) error

	// PublishActionWith 发送虚拟节点的Action消息。指定QOS参数。
	PublishActionWith(virtualId string, data []byte, eventId int64, qos uint8, retained bool) error
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
	mqttRef         mqtt.Client
	mqttEventTopic  string // MQTT使用的EventTopic
	mqttValueTopic  string // MQTT使用的ValueTopic
	mqttActionTopic string // MQTT使用的ActionTopic

	// Shutdown
	stopContext context.Context
	stopCancel  context.CancelFunc
}

func (t *trigger) NodeId() string {
	return t.nodeId
}

func (t *trigger) GenerateEventId() int64 {
	return t.seqIdRef.GenInt64ID()
}

func (t *trigger) NewMessageBy(virtualId string, body []byte, eventId int64) Message {
	return NewMessageWith(t.nodeId, virtualId, body, eventId)
}

func (t *trigger) NewMessageOf(virtualNodeId string, body []byte, eventId int64) Message {
	return NewMessageById(virtualNodeId, body, eventId)
}

func (t *trigger) Startup() {
	t.stopContext, t.stopCancel = context.WithCancel(context.Background())
	// 重建Topic前缀
	t.mqttEventTopic = TopicOfEvents(t.opts.Topic)
	t.mqttValueTopic = TopicOfValues(t.opts.Topic)
	t.mqttActionTopic = TopicOfActions(t.nodeId) // Action使用当前节点作为子Topic
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

func (t *trigger) PublishEvent(virtualId string, data []byte, eventId int64) error {
	return t.PublishEventWith(virtualId, data, eventId, t.globals.MqttQoS, t.globals.MqttRetained)
}

func (t *trigger) PublishEventWith(virtualId string, data []byte, eventId int64, qos uint8, retained bool) error {
	return t.publishMessage(t.mqttEventTopic, virtualId, eventId, data, qos, retained)
}

func (t *trigger) PublishValue(virtualId string, data []byte, eventId int64) error {
	return t.PublishValueWith(virtualId, data, eventId, t.globals.MqttQoS, t.globals.MqttRetained)
}

func (t *trigger) PublishValueWith(virtualId string, data []byte, eventId int64, qos uint8, retained bool) error {
	return t.publishMessage(t.mqttValueTopic, virtualId, eventId, data, qos, retained)
}

func (t *trigger) PublishAction(virtualId string, data []byte, eventId int64) error {
	return t.PublishActionWith(virtualId, data, eventId, t.globals.MqttQoS, t.globals.MqttRetained)
}

func (t *trigger) PublishActionWith(virtualId string, data []byte, eventId int64, qos uint8, retained bool) error {
	return t.publishMessage(t.mqttActionTopic, virtualId, eventId, data, qos, retained)
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

func (t *trigger) publishMessage(mqttTopic string, virtualId string, eventId int64, data []byte, qos byte, retained bool) error {
	return t.PublishMqtt(
		mqttTopic,
		NewMessageWith(t.nodeId, virtualId, data, eventId),
		qos, retained)
}

func (t *trigger) Shutdown() {
	t.mqttRef.Unsubscribe(t.mqttValueTopic, t.mqttEventTopic, t.mqttActionTopic)
	t.stopCancel()
}

func (t *trigger) checkReady() {
	if t.stopCancel == nil || t.stopContext == nil {
		log.Panic("Trigger未启动，须调用Startup()/Shutdown()")
	}
}
