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

	// SendEventMessage 发送事件消息。指定 virtualNodeId 的数据体。
	SendEventMessage(virtualNodeId string, data []byte) error

	// NextSequenceId 返回流水号
	NextSequenceId() uint32

	// 基于内部流水号创建消息对象
	NextMessage(virtualNodeId string, body []byte) Message

	// 发布Inspect消息
	PublishInspectMessage(node MainNode)
}

type TriggerOptions struct {
	NodeName        string          // 节点名称
	Topic           string          // 触发器发送事件的主题
	AutoInspectFunc func() MainNode // Inspect消息生成函数
}

//// trigger

type trigger struct {
	Trigger
	globals    *Globals
	topic      string // Trigger产生的事件Topic
	nodeName   string // Trigger的名称
	sequenceId uint32 // Trigger产生的消息ID序列
	// MainNode 消息生产函数
	autoInspectFunc func() MainNode
	// MQTT
	refMqttClient mqtt.Client
	mqttTopic     string
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
	// 定时发送Inspect消息
	if nil != t.autoInspectFunc {
		go scheduleSendInspect(t.shutdownContext, func() {
			t.PublishInspectMessage(t.autoInspectFunc())
		})
	}
}

func (t *trigger) PublishInspectMessage(node MainNode) {
	mqttSendInspectMessage(t.refMqttClient, t.nodeName, node)
}

func (t *trigger) SendEventMessage(virtualNodeId string, data []byte) error {
	// 构建完整的设备名称
	token := t.refMqttClient.Publish(
		t.mqttTopic,
		t.globals.MqttQoS,
		t.globals.MqttRetained,
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
