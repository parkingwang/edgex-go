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

func mqttSendInspectMessage(client mqtt.Client, nodeName string, inspectFunc func() Inspect) {
	gRpcAddr, addrOK := os.LookupEnv("GRPC_DEVICE_ADDR")
	inspectMsg := inspectFunc()
	for i, vd := range inspectMsg.VirtualDevices {
		// 更新每个设备的gRpc地址、子设备命名
		if addrOK {
			vd.Address = gRpcAddr
		}
		vd.Name = nodeName + ":" + vd.Name
		checkNameFormat(vd.Name)
		// !!修改操作，非引用。注意更新。
		inspectMsg.VirtualDevices[i] = vd
	}
	data, err := json.Marshal(inspectMsg)
	if nil != err {
		log.Panic("Inspect数据序列化错误", err)
	}
	log.Debug("发送DeviceInspect消息, GRPC_DEVICE_ADDR: " + gRpcAddr + ", DATA: " + string(data))
	token := client.Publish(
		tDevicesInspect,
		0,
		true,
		NewMessage([]byte(nodeName), data).getFrames(),
	)
	if token.Wait() && nil != token.Error() {
		log.Error("发送Inspect消息出错", token.Error())
	}
}

func mqttAsyncTickInspect(shutdown context.Context, inspectTask func()) {
	// 在1分钟内上报Inspect消息
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	tick := 1
	for {
		select {
		case <-ticker.C:
			inspectTask()
			tick++
			if tick >= 12 {
				return
			}

		case <-shutdown.Done():
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
