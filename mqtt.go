package edgex

import (
	"encoding/json"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"os"
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

func mqttSendInspectMessage(client mqtt.Client, deviceName string, inspectFunc func() Inspect) {
	log.Debug("发送Inspect消息")
	inspect := inspectFunc()
	gRpcAddr := os.Getenv("GRPC_DEVICE_ADDR")
	for _, i := range inspect.Devices {
		if "" == i.Address {
			i.Address = gRpcAddr
		}
		checkNameFormat(i.Name)
	}
	data, err := json.Marshal(inspect)
	if nil != err {
		log.Panic("Inspect数据错误", err)
	}
	token := client.Publish(
		tDevicesInspect,
		0,
		true,
		NewMessage([]byte(deviceName), data).getFrames(),
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
