package edgex

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"github.com/beinan/fastid"
	"github.com/eclipse/paho.mqtt.golang"
	"sync"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

var (
	ErrExecuteTimeout = errors.New("DRIVER_EXEC_TIMEOUT")
)

type DriverHandler func(topic string, event Message)

type Driver interface {
	NeedLifecycle
	NeedAccessNodeId
	NeedCreateMessages

	// Process 处理消息
	Process(DriverHandler)

	// 发送MQTT消息
	PublishMqtt(mqttTopic string, message Message, qos uint8, retained bool) error

	// 发送Event消息
	PublishEvent(topic string, message Message) error

	// 发送Value消息
	PublishValue(topic string, message Message) error

	// 发送Stats消息
	PublishStatistics(data []byte) error

	// Execute 发起一个同步消息请求
	Execute(remoteNodeId string, in Message, timeout time.Duration) (out Message, err error)

	// Call 发起一个异步请求
	Call(remoteNodeId string, in Message, future chan<- Message) error
}

type DriverOptions struct {
	EventTopics  []string // Event主题列表
	ValueTopics  []string // Value主题列表
	CustomTopics []string // 其它Topic
}

//// Driver实现

type driver struct {
	Driver
	globals  *Globals
	nodeId   string
	opts     DriverOptions
	seqIdRef *fastid.Config
	// Stats
	statistics       *statistics
	statisticsTicker *time.Ticker
	statisticsTopic  string
	// MQTT
	mqttRef mqtt.Client
	// Topics
	mqttListenTopics map[string]byte
	mqttListenWorker DriverHandler
	router           *router
	// shutdown
	stopContext context.Context
	stopCancel  context.CancelFunc
}

func (d *driver) NodeId() string {
	return d.nodeId
}

func (d *driver) NextMessageSequenceId() int64 {
	return d.seqIdRef.GenInt64ID()
}

func (d *driver) NextMessageBy(virtualId string, body []byte) Message {
	return NewMessageWith(d.nodeId, virtualId, body, d.NextMessageSequenceId())
}

func (d *driver) NextMessageOf(virtualNodeId string, body []byte) Message {
	return NewMessageById(virtualNodeId, body, d.NextMessageSequenceId())
}

func (d *driver) Startup() {
	d.stopContext, d.stopCancel = context.WithCancel(context.Background())
	// 监听所有Trigger的UserTopic
	d.mqttListenTopics = make(map[string]byte)
	for _, t := range d.opts.EventTopics {
		eTopic := TopicOfEvents(t)
		d.mqttListenTopics[eTopic] = 0
	}
	for _, t := range d.opts.ValueTopics {
		vTopic := TopicOfValues(t)
		d.mqttListenTopics[vTopic] = 0
	}
	for _, t := range d.opts.CustomTopics {
		d.mqttListenTopics[t] = 0
	}
	for t := range d.mqttListenTopics {
		log.Info("订阅MQTT事件，QoS= 0, Topic= " + t)
	}
	mulToken := d.mqttRef.SubscribeMultiple(d.mqttListenTopics, func(cli mqtt.Client, msg mqtt.Message) {
		frame := msg.Payload()
		if ok, err := CheckMessage(frame); !ok {
			log.Error("消息格式异常", err)
		} else {
			topic := unwrapEdgeXTopic(msg.Topic())
			d.mqttListenWorker(topic, ParseMessage(frame))
		}
	})
	if mulToken.Wait() && nil != mulToken.Error() {
		log.Panic("订阅事件出错", mulToken.Error())
	}
	// 接收Replies
	d.router = &router{
		lock:       new(sync.Mutex),
		registries: make(map[string]*list.List),
	}
	subToken := d.mqttRef.Subscribe(
		topicOfRepliesListen(d.nodeId), 0,
		func(cli mqtt.Client, msg mqtt.Message) {
			if d.globals.LogVerbose {
				log.Debug("接收到RPC响应，Topic= " + msg.Topic())
			}
			d.router.dispatch(msg)
		})
	if subToken.Wait() && nil != subToken.Error() {
		log.Panic("订阅RPC事件出错", subToken.Error())
	}
	// 发送Stats
	d.statistics = new(statistics)
	d.statisticsTicker = time.NewTicker(time.Second * 10)
	d.statisticsTopic = TopicOfStatistics(d.nodeId)
	go func() {
		for {
			select {
			case <-d.statisticsTicker.C:
				err := d.PublishStatistics(d.statistics.toJSONString())
				if nil != err {
					log.Error("定时上报Stats消息，MQTT错误", err)
				}

			case <-d.stopContext.Done():
				return
			}
		}
	}()
}

func (d *driver) Shutdown() {
	topics := make([]string, 0)
	for t := range d.mqttListenTopics {
		topics = append(topics, t)
		log.Info("取消订阅, Topic= ", t)
	}
	token := d.mqttRef.Unsubscribe(topics...)
	if token.Wait() && nil != token.Error() {
		log.Error("取消订阅出错：", token.Error())
	}
	// shutdown
	d.stopCancel()
}

func (d *driver) Process(f DriverHandler) {
	d.mqttListenWorker = f
}

func (d *driver) Call(remoteNodeId string, in Message, future chan<- Message) error {
	d.checkReady()
	log.Debug(" MQ_RPC调用，Endpoint.NodeId= ", remoteNodeId)
	// 发送Request消息
	token := d.mqttRef.Publish(
		topicOfRequestSend(remoteNodeId, d.nodeId),
		d.globals.MqttQoS, false,
		in.Bytes())
	if token.Wait() && nil != token.Error() {
		return token.Error()
	}
	// 通过MQTT的Router来过滤订阅事件
	topic := topicOfRepliesFilter(remoteNodeId, d.nodeId)
	// 过滤相同ID的消息
	matchSeqId := in.SequenceId()
	d.router.register(topic, future, func(msg Message) bool {
		return matchSeqId == msg.SequenceId()
	})
	return nil
}

func (d *driver) Execute(remoteNodeId string, in Message, timeout time.Duration) (Message, error) {
	output := make(chan Message, 1)
	err := d.Call(remoteNodeId, in, output)
	if nil != err {
		return nil, err
	}
	select {
	case msg := <-output:
		return msg, nil

	case <-time.After(timeout):
		return nil, ErrExecuteTimeout
	}
}

func (d *driver) PublishEvent(topic string, msg Message) error {
	return d.PublishMqtt(
		TopicOfEvents(topic),
		msg,
		d.globals.MqttQoS, d.globals.MqttRetained)
}

func (d *driver) PublishValues(topic string, msg Message) error {
	return d.PublishMqtt(
		TopicOfValues(topic),
		msg,
		d.globals.MqttQoS, d.globals.MqttRetained)
}

func (d *driver) PublishStatistics(data []byte) error {
	return d.PublishMqtt(
		d.statisticsTopic,
		d.NextMessageBy(d.nodeId, data),
		0, false)
}

func (d *driver) PublishMqtt(mqttTopic string, msg Message, qos uint8, retained bool) error {
	d.checkReady()
	token := d.mqttRef.Publish(
		mqttTopic,
		qos, retained,
		msg.Bytes())
	token.Wait()
	return token.Error()
}

func (d *driver) checkReady() {
	if d.stopCancel == nil || d.stopContext == nil {
		log.Panic("Driver未启动，须调用Startup()/Shutdown()")
	}
}

////

type statistics struct {
	uptime    time.Time
	recvCount int64
	recvBytes int64
}

func (s *statistics) up() {
	s.uptime = time.Now()
}

func (s *statistics) updateRecv(size int64) {
	s.recvCount++
	s.recvBytes += size
}

func (s *statistics) toJSONString() []byte {
	du := time.Now().Sub(s.uptime).Seconds()
	return []byte(fmt.Sprintf(`{
  "uptime": %d, 
  "recvCount": %d,
  "recvBytes": %d
}`, int64(du), s.recvCount, s.recvBytes))
}

////////

type registry struct {
	future    chan<- Message
	predicate func(Message) bool
	missed    int
}

func (r *registry) invoke(msg Message) bool {
	if r.predicate(msg) {
		r.future <- msg
		return true
	} else {
		r.missed++
		return r.missed >= 10
	}
}

// 消息路由
type router struct {
	lock       *sync.Mutex
	registries map[string]*list.List
}

func (r *router) dispatch(mqttMsg mqtt.Message) {
	r.lock.Lock()
	defer r.lock.Unlock()
	topic := mqttMsg.Topic()
	rl, ok := r.registries[topic]
	if !ok {
		return
	}
	msg := ParseMessage(mqttMsg.Payload())
	for e := rl.Front(); e != nil; e = e.Next() {
		reg := e.Value.(*registry)
		if reg.invoke(msg) {
			rl.Remove(e)
		}
	}
}

func (r *router) register(topic string, future chan<- Message, predicate func(Message) bool) {
	r.lock.Lock()
	defer r.lock.Unlock()
	reg := &registry{
		future:    future,
		predicate: predicate,
		missed:    0,
	}
	l, ok := r.registries[topic]
	if !ok {
		l = list.New()
		r.registries[topic] = l
	}
	l.PushBack(reg)
}
