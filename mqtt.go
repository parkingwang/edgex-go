package edgex

import (
	"context"
	"encoding/json"
	"github.com/eclipse/paho.mqtt.golang"
	"os"
	"runtime"
	"strings"
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
	gRpcAddr, addrOK := os.LookupEnv(EnvKeyGrpcAddress)
	inspectMsg := inspectFunc()
	// 自动更新虚拟设备的参数
	if "" == inspectMsg.HostOS {
		inspectMsg.HostOS = runtime.GOOS
	}
	if "" == inspectMsg.HostArch {
		inspectMsg.HostArch = runtime.GOARCH
	}
	// 更新设备列表参数
	for i, vd := range inspectMsg.VirtualDevices {
		// gRpc地址
		if addrOK {
			vd.Address = gRpcAddr
		}
		// 虚拟设备完整的名称
		checkNameFormat(vd.VirtualName)
		if strings.HasPrefix(vd.VirtualName, nodeName) {
			log.Panic("VirtualName不得以节点名称(NodeName)作为前缀")
		} else {
			vd.VirtualName = CreateVirtualDeviceName(nodeName, vd.VirtualName)
		}
		// !!修改操作，非引用。注意更新。
		inspectMsg.VirtualDevices[i] = vd
	}
	data, err := json.Marshal(inspectMsg)
	if nil != err {
		log.Panic("Inspect数据序列化错误", err)
	}
	// 发送Inspect消息，其中消息来源为NodeName
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
