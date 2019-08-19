package edgex

import (
	"context"
	"github.com/beinan/fastid"
	"github.com/eclipse/paho.mqtt.golang"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// Endpoint是接收、处理，并返回结果的可控制终端节点。
type Endpoint interface {
	NeedLifecycle
	NeedAccessNodeId
	NeedCreateMessages
	NeedProperties

	// 发送MQTT消息，使用原生MQTT Topic来发送
	PublishMqtt(mqttTopic string, message Message, qos uint8, retained bool) error

	// PublishAction 发送虚拟节点的Action发送消息的QoS使用默认设置。
	PublishAction(virtualId string, data []byte, eventId int64) error

	// PublishActionWith 发送虚拟节点的Action消息。指定QOS参数。
	PublishActionWith(virtualId string, data []byte, eventId int64, qos uint8, retained bool) error

	// 处理RPC消息，返回处理结果
	Serve(func(in Message) (out []byte))
}

type EndpointOptions struct {
	NodePropertiesFunc func() MainNodeProperties // // Inspect消息生成函数
}

//// Endpoint实现

type endpoint struct {
	Endpoint
	nodeId     string
	opts       EndpointOptions
	globals    *Globals
	eventIdRef *fastid.Config
	// Rpc
	rpcHandler func(in Message) (out []byte)
	// MQTT
	mqttRef         mqtt.Client
	mqttActionTopic string // MQTT使用的ActionTopic
	mqttRpcTopic    string // MQTT使用的RpcTopic
	// Shutdown
	stopContext context.Context
	stopCancel  context.CancelFunc
}

func (e *endpoint) NodeId() string {
	return e.nodeId
}

func (e *endpoint) GenerateEventId() int64 {
	return e.eventIdRef.GenInt64ID()
}

func (e *endpoint) NewMessageBy(virtualId string, body []byte, eventId int64) Message {
	return NewMessageWith(e.nodeId, virtualId, body, eventId)
}

func (e *endpoint) NewMessageOf(virtualNodeId string, body []byte, eventId int64) Message {
	return NewMessageById(virtualNodeId, body, eventId)
}

func (e *endpoint) PublishAction(virtualId string, data []byte, eventId int64) error {
	return e.PublishActionWith(virtualId, data, eventId, e.globals.MqttQoS, e.globals.MqttRetained)
}

func (e *endpoint) PublishActionWith(virtualId string, data []byte, eventId int64, qos uint8, retained bool) error {
	return e.publishMessage(e.mqttActionTopic, virtualId, eventId, data, qos, retained)
}

func (e *endpoint) PublishMqtt(mqttTopic string, message Message, qos uint8, retained bool) error {
	e.checkReady()
	token := e.mqttRef.Publish(
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

func (e *endpoint) publishMessage(mqttTopic string, virtualId string, eventId int64, data []byte, qos byte, retained bool) error {
	return e.PublishMqtt(
		mqttTopic,
		NewMessageWith(e.nodeId, virtualId, data, eventId),
		qos, retained)
}

func (e *endpoint) Startup() {
	e.stopContext, e.stopCancel = context.WithCancel(context.Background())
	// 监听Endpoint异步RPC事件
	qos := e.globals.MqttQoS
	e.mqttActionTopic = TopicOfActions(e.nodeId) // Action使用当前节点作为子Topic
	e.mqttRpcTopic = topicOfRequestListen(e.nodeId)

	log.Debugf("订阅RPCTopic= %s", e.mqttRpcTopic)
	e.mqttRef.Subscribe(e.mqttRpcTopic, qos, func(cli mqtt.Client, msg mqtt.Message) {
		callerNodeId := topicToRequestCaller(msg.Topic())
		input := ParseMessage(msg.Payload())
		inVnId := input.VirtualNodeId()
		eventId := input.EventId()
		if e.globals.LogVerbose {
			log.Debugf("接收到RPC控制指令，目标：%s, 来源： %s, 事件号：%d",
				inVnId, callerNodeId, eventId)
		}
		body := e.rpcHandler(input)
		// 确保EventId，与Input的相同
		output := NewMessageById(inVnId, body, eventId)
		e.mqttRef.Publish(
			topicOfRepliesSend(e.nodeId, callerNodeId),
			qos, false,
			output.Bytes())
	})
	// 定时发送Properties消息
	if nil != e.opts.NodePropertiesFunc {
		prop := e.opts.NodePropertiesFunc()
		go scheduleSendProperties(e.stopContext, func() {
			e.PublishNodeProperties(prop)
		})
	}
}

func (e *endpoint) PublishNodeProperties(properties MainNodeProperties) {
	e.checkReady()
	properties.NodeId = e.nodeId
	mqttSendNodeProperties(e.globals, e.mqttRef, properties)
}

func (e *endpoint) PublishNodeState(state VirtualNodeState) {
	e.checkReady()
	state.NodeId = e.nodeId
	mqttSendNodeState(e.mqttRef, state)
}

func (e *endpoint) Shutdown() {
	e.mqttRef.Unsubscribe(e.mqttRpcTopic)
	e.stopCancel()
}

func (e *endpoint) Serve(h func(in Message) (out []byte)) {
	e.rpcHandler = h
}

func (e *endpoint) checkReady() {
	if e.stopCancel == nil || e.stopContext == nil {
		log.Panic("Endpoint未启动，须调用Startup()/Shutdown()")
	}
}
