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
	NeedNodeId
	NeedMessages
	NeedInspect

	// 处理RPC消息，返回处理结果
	Serve(func(in Message) (out []byte))
}

type EndpointOptions struct {
	AutoInspectFunc func() MainNodeInfo // // Inspect消息生成函数
}

//// Endpoint实现

type endpoint struct {
	Endpoint
	nodeId     string
	globals    *Globals
	sequenceId uint32
	// MainNodeInfo
	autoInspectFunc func() MainNodeInfo
	// Rpc
	rpcHandler func(in Message) (out []byte)
	// MQTT
	mqttRef mqtt.Client
	// Shutdown
	shutdownContext context.Context
	shutdownCancel  context.CancelFunc
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
	e.shutdownContext, e.shutdownCancel = context.WithCancel(context.Background())
	// 监听Endpoint异步RPC事件
	qos := e.globals.MqttQoS
	listenTopic := topicOfRequestListen(e.nodeId)
	log.Debugf("Mqtt客户端：EndpointListenTopic= %s", listenTopic)
	e.mqttRef.Subscribe(listenTopic, qos, func(cli mqtt.Client, msg mqtt.Message) {
		callerNodeId := topicToRequestCaller(msg.Topic())
		input := ParseMessage(msg.Payload())
		inVnId := input.VirtualNodeId()
		inSeqId := input.SequenceId()
		if e.globals.LogVerbose {
			log.Debugf("接收到控制指令，目标：%s, 来源： %s, 流水号：%d",
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
	// 定时发送Inspect消息
	if nil != e.autoInspectFunc {
		go scheduleSendInspect(e.shutdownContext, func() {
			e.PublishInspect(e.autoInspectFunc())
		})
	}
}

func (e *endpoint) PublishInspect(node MainNodeInfo) {
	mqttSendInspectMessage(e.mqttRef, e.nodeId, node)
}

func (e *endpoint) Shutdown() {
	e.shutdownCancel()
}

func (e *endpoint) Serve(h func(in Message) (out []byte)) {
	e.rpcHandler = h
}
