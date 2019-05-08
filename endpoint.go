package edgex

import (
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// Endpoint是接收、处理，并返回结果的可控制终端节点。
type Endpoint interface {
	Lifecycle
	// 处理消息并返回
	Serve(func(in Packet) (out Packet))
}

type EndpointOptions struct {
	Id   string
	Name string
}

//// Endpoint实现

type endpoint struct {
	Endpoint
	scoped     *GlobalScoped
	topic      string
	endpointId string
	// MQTT
	mqttClient       mqtt.Client
	mqttTopicReply   string
	mqttTopicRequest string
	mqttWorker       func(in Packet) (out Packet)
}

func (e *endpoint) Startup() {
	// 连接Broker
	opts := mqtt.NewClientOptions()
	opts.SetClientID(fmt.Sprintf("Endpoint-%s", e.endpointId))
	opts.AddBroker(e.scoped.MqttBroker)
	opts.SetKeepAlive(e.scoped.MqttKeepAlive)
	opts.SetPingTimeout(e.scoped.MqttPingTimeout)
	opts.SetAutoReconnect(e.scoped.MqttAutoReconnect)
	opts.SetConnectTimeout(e.scoped.MqttConnectTimeout)
	opts.SetWill(topicOfWill("Endpoint", e.endpointId), "offline", 1, true)

	e.mqttClient = mqtt.NewClient(opts)
	log.Info("Mqtt客户端连接Broker: ", e.scoped.MqttBroker)
	if token := e.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Panic("Mqtt客户端连接出错：", token.Error())
	} else {
		log.Info("Mqtt客户端连接成功")
	}

	if e.mqttWorker == nil {
		log.Panic("必须设置消息处理函数: SetReceivedHandler")
	}

	// 开启事件监听
	log.Info("Mqtt客户端Request.Topic: ", e.mqttTopicRequest)
	log.Info("Mqtt客户端Reply.Topic: ", e.mqttTopicReply)
	token := e.mqttClient.Subscribe(e.mqttTopicRequest, e.scoped.MqttQoS, func(cli mqtt.Client, msg mqtt.Message) {
		in := PacketOfBytes(msg.Payload())
		out := e.mqttWorker(in)
		if 0 != len(out) {
			for i := 0; i < 5; i++ {
				if err := e.sendReply(out); nil != err {
					log.Debug("Endpoint重试返回结果: ", i)
					<-time.After(time.Millisecond * 500)
				} else {
					return
				}
			}
		}
	})

	if token.Wait() && nil != token.Error() {
		log.Error("Endpoint订阅操作出错：", token.Error())
	} else {
		log.Info("Endpoint订阅操作成功")
	}
}

func (e *endpoint) Shutdown() {
	token := e.mqttClient.Unsubscribe(e.mqttTopicRequest)
	if token.Wait() && nil != token.Error() {
		log.Error("取消订阅出错：", token.Error())
	}
	e.mqttClient.Disconnect(1000)
}

func (e *endpoint) Serve(w func(in Packet) (out Packet)) {
	e.mqttWorker = w
}

func (e *endpoint) sendReply(data Packet) error {
	token := e.mqttClient.Publish(
		e.mqttTopicReply,
		e.scoped.MqttQoS,
		e.scoped.MqttRetained,
		data.Bytes())
	if token.Wait() && nil != token.Error() {
		return errors.WithMessage(token.Error(), "发送消息出错")
	} else {
		return nil
	}
}
