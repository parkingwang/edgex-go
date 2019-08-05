package edgex

import (
	"context"
	"github.com/eclipse/paho.mqtt.golang"
	"math"
	"strings"
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
	Serve(func(in Message) (out Message))
}

type EndpointOptions struct {
	AutoInspectFunc func() MainNodeInfo // // Inspect消息生成函数
}

//// Endpoint实现

type endpoint struct {
	Endpoint
	nodeId          string
	globals         *Globals
	sequenceId      uint32
	serialExecuting bool
	// MainNodeInfo
	autoInspectFunc func() MainNodeInfo
	handler         func(in Message) (out Message)
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
	log.Debugf("监听控制指令Topic: %s", listenTopic)
	e.mqttRef.Subscribe(listenTopic, qos, func(cli mqtt.Client, msg mqtt.Message) {
		// 控制指令的Topic格式，已由Subscription确保其格式：
		// $EdgeX/requests/ MyNodeId / SeqId / Remote调用者节点ID
		callerNodeId := strings.Split(msg.Topic(), "/")[4]
		input := ParseMessage(msg.Payload())
		vnId := input.VirtualNodeId()
		log.Debugf("接收到控制指令，目标：%s, 来源： %s", vnId, callerNodeId)
		e.mqttRef.Publish(topicOfRepliesSend(e.nodeId, input.SequenceId(), callerNodeId),
			qos, false,
			e.handler(input).Bytes())
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

func (e *endpoint) Serve(h func(in Message) (out Message)) {
	e.handler = h
}
