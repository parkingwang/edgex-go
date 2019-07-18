package edgex

import (
	"context"
	"fmt"
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
}

type TriggerOptions struct {
	NodeName        string          // 节点名称
	Topic           string          // 触发器发送事件的主题
	InspectNodeFunc func() MainNode // Inspect消息生成函数
}

//// trigger

type trigger struct {
	Trigger
	globals    *Globals
	topic      string // Trigger产生的事件Topic
	nodeName   string // Trigger的名称
	sequenceId uint32 // Trigger产生的消息ID序列
	// MainNode 消息生产函数
	inspectNodeFunc func() MainNode
	// MQTT
	mqttClient mqtt.Client
	mqttTopic  string
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
	// 连接Broker
	opts := mqtt.NewClientOptions()
	clientId := fmt.Sprintf("EX-Trigger:%s", t.nodeName)
	opts.SetClientID(clientId)
	opts.SetWill(topicOfOffline("Trigger", t.nodeName),
		"offline", 1, true)
	mqttSetOptions(opts, t.globals)
	t.mqttTopic = topicOfTriggerEvents(t.topic)
	t.mqttClient = mqtt.NewClient(opts)
	log.Info("Mqtt客户端连接Broker: ", t.globals.MqttBroker)

	// 连续重试
	mqttAwaitConnection(t.mqttClient, t.globals.MqttMaxRetry)

	if !t.mqttClient.IsConnected() {
		log.Panic("Mqtt客户端连接无法连接Broker")
	} else {
		log.Debug("Mqtt客户端连接成功：" + clientId)
		// 异步发送Inspect消息
		mqttSendInspectMessage(t.mqttClient, t.nodeName, t.inspectNodeFunc)
		// 定时发送
		go mqttAsyncTickInspect(t.shutdownContext, func() {
			mqttSendInspectMessage(t.mqttClient, t.nodeName, t.inspectNodeFunc)
		})
	}
}

func (t *trigger) SendEventMessage(virtualNodeId string, data []byte) error {
	// 构建完整的设备名称
	token := t.mqttClient.Publish(
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
	t.mqttClient.Disconnect(t.globals.MqttQuitMillSec)
}
