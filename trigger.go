package edgex

import (
	"context"
	"github.com/bwmarrin/snowflake"
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
	PublishEvent(groupId, majorId, minorId string, data []byte, eventId int64) error

	// PublishEventMessage 发送虚拟节点的Event消息。
	PublishEventMessage(message Message) error

	// PublishValue 发送虚拟节点的Value消息。发送消息的QoS使用默认设置。
	PublishValue(groupId, majorId, minorId string, data []byte, eventId int64) error

	// PublishValueMessage 发送虚拟节点的Value消息
	PublishValueMessage(message Message) error
}

type TriggerOptions struct {
	Topic              string                    // 触发器发送事件的主题
	NodePropertiesFunc func() MainNodeProperties // Inspect消息生成函数
}

//// trigger

type trigger struct {
	Trigger
	nodeId     string // Trigger的名称
	opts       TriggerOptions
	globals    *Globals
	eventIdRef *snowflake.Node // Trigger产生的消息ID序列
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
	return t.eventIdRef.Generate().Int64()
}

func (t *trigger) NewMessage(groupId, majorId, minorId string, body []byte, eventId int64) Message {
	return NewMessage(t.nodeId, groupId, majorId, minorId, body, eventId)
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

func (t *trigger) PublishEvent(groupId, majorId, minorId string, data []byte, eventId int64) error {
	return t.PublishEventMessage(t.NewMessage(groupId, majorId, minorId, data, eventId))
}

func (t *trigger) PublishEventMessage(message Message) error {
	return t.PublishMqtt(
		t.mqttEventTopic,
		message,
		t.globals.MqttQoS, t.globals.MqttRetained)
}

func (t *trigger) PublishValue(groupId, majorId, minorId string, data []byte, eventId int64) error {
	return t.PublishValueMessage(t.NewMessage(groupId, majorId, minorId, data, eventId))
}

func (t *trigger) PublishValueMessage(message Message) error {
	return t.PublishMqtt(
		t.mqttValueTopic,
		message,
		t.globals.MqttQoS, t.globals.MqttRetained)
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

func (t *trigger) Shutdown() {
	t.mqttRef.Unsubscribe(t.mqttValueTopic, t.mqttEventTopic, t.mqttActionTopic)
	t.stopCancel()
}

func (t *trigger) checkReady() {
	if t.stopCancel == nil || t.stopContext == nil {
		log.Panic("Trigger未启动，须调用Startup()/Shutdown()")
	}
}
