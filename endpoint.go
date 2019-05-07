package edgex

import (
	"errors"
	"github.com/eclipse/paho.mqtt.golang"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// Endpoint是接收和处理消息的终端节点。它可以接收控制指令
type Endpoint interface {
	Lifecycle
	// 发送消息
	Send(b Frame) error
	// 接收消息
	Recv() (b Frame, err error)
	// 接收消息通道
	RecvChan() <-chan Frame
}

var ErrRecvTimeout = errors.New("receive timeout")

////

type endpoint struct {
	Endpoint
	global *Global
	info   NodeInfo
	mqtt   mqtt.Client

	topicSend string
	topicRecv string
	recvChan  chan Frame
}

func (e *endpoint) Startup(args map[string]interface{}) {
	e.topicSend = TopicEndpointSendQ(e.info.Topic, e.info.Uuid)
	e.topicRecv = TopicEndpointRecvQ(e.info.Topic, e.info.Uuid)
	e.recvChan = make(chan Frame, 2)
	// Mqtt connect
	log.Debugf("Mqtt客户端连接Broker: %s", e.global.MqttBroker)
	opts := mqtt.NewClientOptions().AddBroker(e.global.MqttBroker).SetClientID(e.info.Uuid)
	opts.SetKeepAlive(2 * time.Second)
	opts.SetPingTimeout(1 * time.Second)
	opts.SetAutoReconnect(true)

	e.mqtt = mqtt.NewClient(opts)
	if token := e.mqtt.Connect(); token.Wait() && token.Error() != nil {
		log.Panic("Mqtt客户端连接出错：", token.Error())
	} else {
		log.Info("Mqtt客户端连接成功")
		log.Debugf("Mqtt客户端SendQ.Topic: %s", e.topicSend)
		log.Debugf("Mqtt客户端RecvQ.Topic: %s", e.topicRecv)
	}

	token := e.mqtt.Subscribe(e.topicRecv, 2, func(cli mqtt.Client, msg mqtt.Message) {
		select {
		case e.recvChan <- msg.Payload():
			log.Debugf("接收到消息：%s", msg.Topic())

		default:
			log.Warn("消息队列繁忙")
		}
	})
	if token.Wait() && nil != token.Error() {
		log.Error("事件订阅出错：", token.Error())
	} else {
		log.Debugf("事件订阅成功")
	}
}

func (e *endpoint) Shutdown() {
	token := e.mqtt.Unsubscribe(e.topicRecv)
	if token.Wait() && nil != token.Error() {
		log.Error("取消订阅出错：", token.Error())
	}
}

func (e *endpoint) Send(frames Frame) error {
	token := e.mqtt.Publish(e.topicSend, 2, false, []byte(frames))
	if token.Wait() && nil != token.Error() {
		return errors.New("发送消息出错: " + token.Error().Error())
	} else {
		log.Debugf("发送消息成功: %s", e.topicSend)
		return nil
	}
}

func (e *endpoint) Recv() (Frame, error) {
	select {
	case <-time.After(time.Second):
		return nil, ErrRecvTimeout

	case b := <-e.recvChan:
		return b, nil
	}
}

func (e *endpoint) RecvChan() <-chan Frame {
	return e.recvChan
}
