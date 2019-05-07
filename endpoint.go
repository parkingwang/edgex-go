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
	global   *GlobalScoped
	topic    string
	id       string
	recvChan chan Frame
	// MQTT
	mqtt          mqtt.Client
	mqttTopicSend string
	mqttTopicRecv string
}

func (e *endpoint) Startup(args map[string]interface{}) {
	// 连接Broker
	opts := mqtt.NewClientOptions().AddBroker(e.global.MqttBroker)
	opts.SetClientID(e.id)
	opts.SetKeepAlive(e.global.MqttKeepAlive)
	opts.SetPingTimeout(e.global.MqttPingTimeout)
	opts.SetAutoReconnect(e.global.MqttAutoReconnect)
	opts.SetConnectTimeout(e.global.MqttConnectTimeout)
	e.mqtt = mqtt.NewClient(opts)
	log.Debugf("Mqtt客户端连接Broker: %s", e.global.MqttBroker)
	if token := e.mqtt.Connect(); token.Wait() && token.Error() != nil {
		log.Panic("Mqtt客户端连接出错：", token.Error())
	} else {
		log.Info("Mqtt客户端连接成功")
		log.Debugf("Mqtt客户端SendQ.Topic: %s", e.mqttTopicSend)
		log.Debugf("Mqtt客户端RecvQ.Topic: %s", e.mqttTopicRecv)
	}

	// 开启Recv订阅事件
	sub := e.mqtt.Subscribe(e.mqttTopicRecv, e.global.MqttQoS, func(cli mqtt.Client, msg mqtt.Message) {
		select {
		case e.recvChan <- msg.Payload():
			log.Debugf("接收到消息：%s", msg.Topic())

		default:
			log.Warn("消息队列繁忙")
		}
	})
	if sub.Wait() && nil != sub.Error() {
		log.Error("事件订阅出错：", sub.Error())
	} else {
		log.Debugf("事件订阅成功")
	}
}

func (e *endpoint) Shutdown() {
	unsub := e.mqtt.Unsubscribe(e.mqttTopicRecv)
	if unsub.Wait() && nil != unsub.Error() {
		log.Error("取消订阅出错：", unsub.Error())
	}
	e.mqtt.Disconnect(1000)
}

func (e *endpoint) Send(frames Frame) error {
	token := e.mqtt.Publish(e.mqttTopicSend, e.global.MqttQoS, e.global.MqttRetained, []byte(frames))
	if token.Wait() && nil != token.Error() {
		return errors.New("发送消息出错: " + token.Error().Error())
	} else {
		log.Debugf("发送消息成功: %s", e.mqttTopicSend)
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
