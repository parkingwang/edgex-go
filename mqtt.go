package edgex

import (
	"encoding/json"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

func mqttSetOptions(opts *mqtt.ClientOptions, scoped *GlobalScoped) {
	opts.AddBroker(scoped.MqttBroker)
	opts.SetKeepAlive(scoped.MqttKeepAlive)
	opts.SetPingTimeout(scoped.MqttPingTimeout)
	opts.SetAutoReconnect(scoped.MqttAutoReconnect)
	opts.SetConnectTimeout(scoped.MqttConnectTimeout)
	opts.SetCleanSession(scoped.MqttCleanSession)
	opts.SetMaxReconnectInterval(scoped.MqttReconnectInterval)
}

////

func mqttSendInspectMessage(client mqtt.Client, inspectFunc func() Inspect) {
	inspect, err := json.Marshal(inspectFunc())
	if nil != err {
		log.Panic("Inspect数据错误", err)
	}
	token := client.Publish(
		tDevicesInspect,
		0,
		true,
		inspect,
	)
	if token.Wait() && nil != token.Error() {
		log.Panic("发送Inspect消息出错", token.Error())
	}
}

func mqttSendAliveMessage(client mqtt.Client, typeName, devName string, alive Message) error {
	token := client.Publish(
		topicOfAlive(typeName, devName),
		0,
		true,
		alive.getFrames())
	if token.Wait() && nil != token.Error() {
		return errors.WithMessage(token.Error(), "发送Alive消息出错")
	} else {
		return nil
	}
}

func mqttRetryConnect(client mqtt.Client, max int) {
	for retry := 0; retry < max; retry++ {
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			log.Error("Mqtt客户端连接出错：", token.Error())
			<-time.After(time.Second)
		} else {
			log.Info("Mqtt客户端连接成功")
			break
		}
	}
}
