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

//// Endpoint实现

type endpoint struct {
	Endpoint
	scoped   *GlobalScoped
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
	opts := mqtt.NewClientOptions()
	opts.SetClientID(e.id)
	opts.AddBroker(e.scoped.MqttBroker)
	opts.SetKeepAlive(e.scoped.MqttKeepAlive)
	opts.SetPingTimeout(e.scoped.MqttPingTimeout)
	opts.SetAutoReconnect(e.scoped.MqttAutoReconnect)
	opts.SetConnectTimeout(e.scoped.MqttConnectTimeout)
	e.mqtt = mqtt.NewClient(opts)
	log.Info("Mqtt客户端连接Broker: ", e.scoped.MqttBroker)
	if token := e.mqtt.Connect(); token.Wait() && token.Error() != nil {
		log.Panic("Mqtt客户端连接出错：", token.Error())
	} else {
		log.Info("Mqtt客户端连接成功")
		log.Info("Mqtt客户端SendQ.Topic: ", e.mqttTopicSend)
		log.Info("Mqtt客户端RecvQ.Topic: ", e.mqttTopicRecv)
	}

	// 开启Recv订阅事件
	token := e.mqtt.Subscribe(e.mqttTopicRecv, e.scoped.MqttQoS,
		func(cli mqtt.Client, msg mqtt.Message) {
			select {
			case e.recvChan <- msg.Payload():
				log.Debug("接收到消息：%s", msg.Topic())

			default:
				log.Warn("消息队列繁忙")
			}
		})
	if token.Wait() && nil != token.Error() {
		log.Error("事件订阅出错：", token.Error())
	} else {
		log.Info("事件订阅成功")
	}
}

func (e *endpoint) Shutdown() {
	token := e.mqtt.Unsubscribe(e.mqttTopicRecv)
	if token.Wait() && nil != token.Error() {
		log.Error("取消订阅出错：", token.Error())
	}
	e.mqtt.Disconnect(1000)
}

func (e *endpoint) Send(frames Frame) error {
	token := e.mqtt.Publish(e.mqttTopicSend, e.scoped.MqttQoS, e.scoped.MqttRetained, []byte(frames))
	if token.Wait() && nil != token.Error() {
		return errors.New("发送消息出错: " + token.Error().Error())
	} else {
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
