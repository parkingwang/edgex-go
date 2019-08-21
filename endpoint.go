package edgex

import (
	"context"
	"github.com/bwmarrin/snowflake"
	"github.com/eclipse/paho.mqtt.golang"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// 指令处理函数，返回两个结果：1. 处理结果；2. 动作消息
type EndpointServeHandler func(request Message) (response []byte)

// Endpoint是接收、处理，并返回结果的可控制终端节点。
type Endpoint interface {
	NeedLifecycle
	NeedAccessNodeId
	NeedCreateMessages
	NeedProperties

	// 发送MQTT消息，使用原生MQTT Topic来发送
	PublishMqtt(mqttTopic string, message Message, qos uint8, retained bool) error

	// PublishAction 发送虚拟节点的Action发送消息的QoS使用默认设置。
	PublishAction(groupId, majorId, minorId string, data []byte, eventId int64) error

	// PublishActionMessage 发送虚拟节点的Action发送消息的QoS使用默认设置。
	PublishActionMessage(message Message) error

	// 处理RPC消息，返回处理结果及Action
	Serve(handler EndpointServeHandler)
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
	eventIdRef *snowflake.Node
	// Rpc
	rpcServeHandler EndpointServeHandler
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
	return e.eventIdRef.Generate().Int64()
}

func (e *endpoint) NewMessage(groupId, majorId, minorId string, body []byte, eventId int64) Message {
	return NewMessage(e.nodeId, groupId, majorId, minorId, body, eventId)
}

func (e *endpoint) PublishAction(groupId, majorId, minorId string, data []byte, eventId int64) error {
	return e.PublishActionMessage(e.NewMessage(groupId, majorId, minorId, data, eventId))
}

func (e *endpoint) PublishActionMessage(message Message) error {
	return e.PublishMqtt(
		e.mqttActionTopic,
		message,
		e.globals.MqttQoS, e.globals.MqttRetained)
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

func (e *endpoint) Startup() {
	e.stopContext, e.stopCancel = context.WithCancel(context.Background())
	// 监听Endpoint异步RPC事件
	qos := e.globals.MqttQoS
	e.mqttActionTopic = TopicOfActions(e.nodeId) // Action使用当前节点作为子Topic
	e.mqttRpcTopic = topicOfRequestListen(e.nodeId)

	log.Debugf("订阅RPC-Topic= %s", e.mqttRpcTopic)
	e.mqttRef.Subscribe(e.mqttRpcTopic, qos, func(cli mqtt.Client, msg mqtt.Message) {
		callerNodeId := topicToRequestCaller(msg.Topic())
		input := ParseMessage(msg.Payload())
		unionId := input.UnionId()
		eventId := input.EventId()
		if e.globals.LogVerbose {
			log.Debugf("接收RPC控制指令，目标：%s, 来源： %s, 事件号：%d",
				unionId, callerNodeId, eventId)
		}
		output := e.rpcServeHandler(input)
		// 确保EventId，与Input的相同
		for i := 0; i <= 5; i++ {
			token := e.mqttRef.Publish(
				topicOfRepliesSend(e.nodeId, callerNodeId),
				qos, false,
				NewMessageByUnionId(unionId, output, eventId).Bytes())
			if token.Wait() && nil != token.Error() {
				log.Error("返回RPC响应出错，正在重试(500ms)：", token.Error())
				<-time.After(500 * time.Millisecond)
			} else {
				break
			}
		}
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

func (e *endpoint) Serve(h EndpointServeHandler) {
	e.rpcServeHandler = h
}

func (e *endpoint) checkReady() {
	if e.stopCancel == nil || e.stopContext == nil {
		log.Panic("Endpoint未启动，须调用Startup()/Shutdown()")
	}
}
