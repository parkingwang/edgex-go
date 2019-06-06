package edgex

import (
	"context"
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
	gRpcAddr, addrOK := os.LookupEnv("GRPC_DEVICE_ADDR")
	inspect := inspectFunc()
	for i, dev := range inspect.Devices {
		// 更新每个设备的gRpc地址
		if addrOK {
			dev.Address = gRpcAddr
			// !!修改操作，非引用。注意更新。
			inspect.Devices[i] = dev
		}
		checkNameFormat(dev.Name)
	}
	data, err := json.Marshal(inspect)
	if nil != err {
		log.Panic("Inspect数据序列化错误", err)
	}
	log.Debug("发送DeviceInspect消息, GRPC_DEVICE_ADDR: " + gRpcAddr + ", DATA: " + string(data))
	token := client.Publish(
		tDevicesInspect,
		0,
		true,
		NewMessage([]byte(deviceName), data).getFrames(),
	)
	if token.Wait() && nil != token.Error() {
		log.Error("发送Inspect消息出错", token.Error())
	}
}

func mqttAsyncTickInspect(context context.Context, inspectTask func()) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	tick := 1
	for {
		select {
		case <-timer.C:
			inspectTask()
			tick += 5
			du := time.Second * time.Duration(tick)
			if du > time.Minute*5 {
				du = time.Minute * 5
			}
			timer.Reset(du)

		case <-context.Done():
			return
		}
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

func mqttAwaitConnection(client mqtt.Client, maxRetry int) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for i := 1; i <= maxRetry; i++ {
		<-timer.C
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			if i == maxRetry {
				log.Errorf("[%d] Mqtt客户端连接失败，最大次数：%v", i, token.Error())
			} else {
				log.Debugf("[%d] Mqtt客户端尝试重新连接，失败：%v", i, token.Error())
			}
			timer.Reset(time.Second * time.Duration(i))
		} else {
			log.Info("Mqtt客户端连接成功")
			break
		}
	}
}
