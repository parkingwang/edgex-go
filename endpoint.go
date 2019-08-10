package edgex

import (
	"context"
	"github.com/eclipse/paho.mqtt.golang"
	"math"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// Endpoint是接收、处理，并返回结果的可控制终端节点。
type Endpoint interface {
	NeedLifecycle
	NeedAccessNodeId
	NeedCreateMessages
	NeedInspectProperties

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
	sequenceId uint32
	// Rpc
	rpcHandler func(in Message) (out []byte)
	// MQTT
	mqttRef mqtt.Client
	// Shutdown
	stopContext context.Context
	stopCancel  context.CancelFunc
}

func (e *endpoint) NodeId() string {
	return e.nodeId
}

func (e *endpoint) NextMessageSequenceId() uint32 {
	e.sequenceId = (e.sequenceId + 1) % math.MaxUint32
	return e.sequenceId
}

func (e *endpoint) NextMessageBy(virtualId string, body []byte) Message {
	return NewMessageWith(e.nodeId, virtualId, body, e.NextMessageSequenceId())
}

func (e *endpoint) NextMessageOf(virtualNodeId string, body []byte) Message {
	return NewMessageById(virtualNodeId, body, e.NextMessageSequenceId())
}

func (e *endpoint) Startup() {
	e.stopContext, e.stopCancel = context.WithCancel(context.Background())
	// 监听Endpoint异步RPC事件
	qos := e.globals.MqttQoS
	rpcTopic := topicOfRequestListen(e.nodeId)
	log.Debugf("Mqtt客户端：Endpoint RPC Topic= %s", rpcTopic)
	e.mqttRef.Subscribe(rpcTopic, qos, func(cli mqtt.Client, msg mqtt.Message) {
		callerNodeId := topicToRequestCaller(msg.Topic())
		input := ParseMessage(msg.Payload())
		inVnId := input.VirtualNodeId()
		inSeqId := input.SequenceId()
		if e.globals.LogVerbose {
			log.Debugf("接收到RPC控制指令，目标：%s, 来源： %s, 流水号：%d",
				inVnId, callerNodeId, inSeqId)
		}
		body := e.rpcHandler(input)
		// 确保SequenceId，与Input的相同
		output := NewMessageById(inVnId, body, inSeqId)
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
