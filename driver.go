package edgex

import (
	"context"
	"errors"
	"fmt"
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

type DriverHandler func(topic string, event Message)

type Driver interface {
	NeedLifecycle
	NeedNodeId
	NeedMessages

	// Process 处理消息
	Process(DriverHandler)

	// 发送MQTT消息
	Publish(mqttTopic string, msg Message, qos uint8, retained bool)

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
	// Stats
	stats       *stats
	statsTicker *time.Ticker
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
	d.stopContext, d.stopCancel = context.WithCancel(context.Background())
	// 监听所有Trigger的UserTopic
	d.mqttListenTopics = make(map[string]byte)
	for _, t := range d.opts.EventTopics {
		eTopic := topicOfEvents(t)
		d.mqttListenTopics[eTopic] = 0
	}
	for _, t := range d.opts.ValueTopics {
		vTopic := topicOfValues(t)
		d.mqttListenTopics[vTopic] = 0
	}
	for _, t := range d.opts.CustomTopics {
		d.mqttListenTopics[t] = 0
	}
	for t, _ := range d.mqttListenTopics {
		log.Info("Mqtt客户端：监听事件，QoS= 0, Topic= " + t)
	}
	mToken := d.mqttRef.SubscribeMultiple(d.mqttListenTopics, func(cli mqtt.Client, msg mqtt.Message) {
		frame := msg.Payload()
		if ok, err := CheckMessage(frame); !ok {
			log.Error("消息格式异常: ", err)
		} else {
			topic := unwrapEdgeXTopic(msg.Topic())
			d.mqttListenWorker(topic, ParseMessage(frame))
		}
	})
	if mToken.Wait() && nil != mToken.Error() {
		log.Error("监听TRIGGER事件出错：", mToken.Error())
	}
	// 接收Replies
	d.router = &router{
		lock:     new(sync.Mutex),
		registry: make(map[string][]routerHandler),
	}
	rToken := d.mqttRef.Subscribe(topicOfRepliesListen(d.nodeId), 0, func(cli mqtt.Client, msg mqtt.Message) {
		if d.globals.LogVerbose {
			log.Debug("接收到RPC响应，Topic: " + msg.Topic())
		}
		d.router.dispatchThenRemove(msg)
	})
	if rToken.Wait() && nil != rToken.Error() {
		log.Error("监听RPC事件出错：", rToken.Error())
	}
	// 发送Stats
	d.stats = new(stats)
	d.statsTicker = time.NewTicker(time.Second * 10)
	go func() {
		mqttTopic := topicOfStats(d.nodeId)
		for {
			select {
			case <-d.statsTicker.C:
				d.publishMqtt(mqttTopic,
					d.NextMessageBy(d.nodeId, d.stats.toJSONString()),
					0,
					false,
				)

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
		log.Info("取消监听事件: ", t)
	}
	token := d.mqttRef.Unsubscribe(topics...)
	if token.Wait() && nil != token.Error() {
		log.Error("取消监听出错：", token.Error())
	}
	// shutdown
	d.stopCancel()
}

func (d *driver) Process(f DriverHandler) {
	d.mqttListenWorker = f
}

func (d *driver) Call(remoteNodeId string, in Message, callback func(out Message, err error)) {
	log.Debug("MQ_RPC调用Endpoint.NodeId: ", remoteNodeId)
	// 发送Request消息
	token := d.mqttRef.Publish(
		topicOfRequestSend(remoteNodeId, d.nodeId),
		d.globals.MqttQoS, false,
		in.Bytes())
	if token.Wait() && nil != token.Error() {
		log.Error("发送MQ_RPC消息出错", token.Error())
		callback(nil, token.Error())
	} else {
		// 通过MQTT的Router来过滤订阅事件
		topic := topicOfRepliesFilter(remoteNodeId, d.nodeId)
		// 过滤相同ID的消息
		d.router.addFilter(topic, func(out Message) bool {
			match := in.SequenceId() == out.SequenceId()
			if match {
				callback(out, nil)
			}
			return match
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

func (d *driver) Publish(mqttTopic string, msg Message, qos uint8, retained bool) {
	d.publishMqtt(mqttTopic, msg, qos, retained)
}

func (d *driver) publishMqtt(mqttTopic string, msg Message, qos uint8, retained bool) {
	token := d.mqttRef.Publish(
		mqttTopic,
		qos, retained,
		msg.Bytes())
	if token.Wait() && nil != token.Error() {
		log.Error("发送MQTT消息出错：", token.Error())
	}
}

////

type stats struct {
	uptime    time.Time
	recvCount int64
	recvBytes int64
}

func (s *stats) up() {
	s.uptime = time.Now()
}

func (s *stats) updateRecv(size int64) {
	s.recvCount++
	s.recvBytes += size
}

func (s *stats) toJSONString() []byte {
	du := time.Now().Sub(s.uptime).Seconds()
	return []byte(fmt.Sprintf(`{
  "uptime": %d, 
  "recvCount": %d,
  "recvBytes": %d
}`, int64(du), s.recvCount, s.recvBytes))
}

////////

type routerHandler func(Message) bool

// 消息路由
type router struct {
	lock     *sync.Mutex
	registry map[string][]routerHandler
}

func (r *router) dispatchThenRemove(mqttMsg mqtt.Message) {
	r.lock.Lock()
	defer r.lock.Unlock()

	topic := mqttMsg.Topic()
	msg := ParseMessage(mqttMsg.Payload())
	if hs, ok := r.registry[topic]; ok {
		remains := make([]routerHandler, 0)
		for _, handler := range hs {
			if !handler(msg) {
				remains = append(remains, handler)
			}
		}
		if 0 == len(remains) {
			delete(r.registry, topic)
		}
	}
}

func (r *router) addFilter(topic string, handler routerHandler) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if hs, ok := r.registry[topic]; ok {
		r.registry[topic] = append(hs, handler)
	} else {
		r.registry[topic] = []routerHandler{handler}
	}

}
