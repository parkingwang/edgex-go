package edgex

import (
	"errors"
	"github.com/eclipse/paho.mqtt.golang"
	"math"
	"sync"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

var (
	ErrExecuteTimeout = errors.New("DRIVER_EXEC_TIMEOUT")
)

type Driver interface {
	NeedLifecycle
	NeedNodeId
	NeedMessages

	// Process 处理消息
	Process(func(event Message))

	// Execute 发起一个同步消息请求
	Execute(remoteNodeId string, in Message, timeout time.Duration) (out Message, err error)

	// Call 发起一个异步请求
	Call(remoteNodeId string, in Message, callback func(out Message, err error))
}

type DriverOptions struct {
	EventTopics  []string // Event主题列表
	ValueTopics  []string // Value主题列表
	CustomTopics []string // 其它Topic
}

//// Driver实现

type driver struct {
	Driver
	globals    *Globals
	nodeId     string
	sequenceId uint32
	opts       DriverOptions
	// MQTT
	mqttRef mqtt.Client
	// Topics
	mqttListenTopics map[string]byte
	mqttListenWorker func(event Message)
	router           *router
}

func (d *driver) NodeId() string {
	return d.nodeId
}

func (d *driver) NextMessageSequenceId() uint32 {
	d.sequenceId = (d.sequenceId + 1) % math.MaxUint32
	return d.sequenceId
}

func (d *driver) NextMessageBy(virtualId string, body []byte) Message {
	return NewMessageWith(d.nodeId, virtualId, body, d.NextMessageSequenceId())
}

func (d *driver) NextMessageOf(virtualNodeId string, body []byte) Message {
	return NewMessageById(virtualNodeId, body, d.NextMessageSequenceId())
}

func (d *driver) Startup() {
	// 监听所有Trigger的UserTopic
	d.mqttListenTopics = make(map[string]byte)
	for _, t := range d.opts.EventTopics {
		eTopic := topicOfEvents(t)
		d.mqttListenTopics[eTopic] = 0
		log.Info("开启监听[Event]: ", eTopic)
	}
	for _, t := range d.opts.ValueTopics {
		vTopic := topicOfValues(t)
		d.mqttListenTopics[vTopic] = 0
		log.Info("开启监听[Value]: ", vTopic)
	}
	for _, t := range d.opts.CustomTopics {
		d.mqttListenTopics[t] = 0
		log.Info("开启监听[Custom]: ", t)
	}
	mToken := d.mqttRef.SubscribeMultiple(d.mqttListenTopics, func(cli mqtt.Client, msg mqtt.Message) {
		frame := msg.Payload()
		if ok, err := CheckMessage(frame); !ok {
			log.Error("消息格式异常: ", err)
		} else {
			d.mqttListenWorker(ParseMessage(frame))
		}
	})
	if mToken.Wait() && nil != mToken.Error() {
		log.Error("监听TRIGGER事件出错：", mToken.Error())
	}
	// 接收Replies
	d.router = &router{
		lock:     new(sync.Mutex),
		handlers: make(map[string]func(mqtt.Message)),
	}
	rToken := d.mqttRef.Subscribe(topicOfRepliesListen(d.nodeId), 0, func(cli mqtt.Client, msg mqtt.Message) {
		log.Debug("接收到RPC响应，Topic: " + msg.Topic())
		d.router.dispatchThenRemove(msg)
	})
	if rToken.Wait() && nil != rToken.Error() {
		log.Error("监听RPC事件出错：", rToken.Error())
	}
}

func (d *driver) Shutdown() {
	topics := make([]string, 0)
	for t := range d.mqttListenTopics {
		topics = append(topics, t)
		log.Info("取消监听事件: ", t)
	}
	token := d.mqttRef.Unsubscribe(topics...)
	if token.Wait() && nil != token.Error() {
		log.Error("取消监听出错：", token.Error())
	}
}

func (d *driver) Process(f func(Message)) {
	d.mqttListenWorker = f
}

func (d *driver) Call(remoteNodeId string, in Message, callback func(out Message, err error)) {
	log.Debug("MQ_RPC调用Endpoint.NodeId: ", remoteNodeId)
	// 发送Request消息
	token := d.mqttRef.Publish(
		topicOfRequestSend(remoteNodeId, in.SequenceId(), d.nodeId),
		d.globals.MqttQoS, false,
		in.Bytes())
	if token.Wait() && nil != token.Error() {
		log.Error("发送MQ_RPC消息出错", token.Error())
		callback(nil, token.Error())
	} else {
		// 通过MQTT的Router来过滤订阅事件
		topic := topicOfRepliesFilter(remoteNodeId, in.SequenceId(), d.nodeId)
		d.router.register(topic, func(msg mqtt.Message) {
			callback(ParseMessage(msg.Payload()), nil)
		})
	}
}

func (d *driver) Execute(remoteNodeId string, in Message, timeout time.Duration) (out Message, err error) {
	outC := make(chan Message, 1)
	errC := make(chan error, 1)
	d.Call(remoteNodeId, in, func(out Message, err error) {
		if nil != err {
			errC <- err
		} else {
			outC <- out
		}
	})
	select {
	case out := <-outC:
		return out, nil

	case err := <-errC:
		return nil, err

	case <-time.After(timeout):
		return nil, ErrExecuteTimeout
	}
}

// 消息路由
type router struct {
	lock     *sync.Mutex
	handlers map[string]func(msg mqtt.Message)
}

func (r *router) dispatchThenRemove(msg mqtt.Message) {
	r.lock.Lock()
	defer r.lock.Unlock()
	t := msg.Topic()
	if handler, ok := r.handlers[t]; ok {
		delete(r.handlers, t)
		handler(msg)
	}
}

func (r *router) register(topic string, handler func(msg mqtt.Message)) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.handlers[topic] = handler
}
