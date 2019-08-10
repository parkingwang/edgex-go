package edgex

import (
	"context"
	"encoding/json"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"runtime"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

func mqttSetOptions(opts *mqtt.ClientOptions, scoped *Globals) {
	opts.AddBroker(scoped.MqttBroker)
	opts.SetKeepAlive(scoped.MqttKeepAlive)
	opts.SetPingTimeout(scoped.MqttPingTimeout)
	opts.SetAutoReconnect(scoped.MqttAutoReconnect)
	opts.SetConnectTimeout(scoped.MqttConnectTimeout)
	opts.SetCleanSession(scoped.MqttCleanSession)
	opts.SetMaxReconnectInterval(scoped.MqttReconnectInterval)
	if "" != scoped.MqttUsername && "" != scoped.MqttPassword {
		opts.Username = scoped.MqttUsername
		opts.Password = scoped.MqttPassword
	}
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Error("Mqtt客户端：丢失连接（" + err.Error() + ")")
	})
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Debug("Mqtt客户端：已连接")
	})
}

////

func createStateMessage(state VirtualNodeState) Message {
	if "" != state.Uuid {
		log.Debugf("虚拟节点使用自定义Uuid：%s", state.Uuid)
	} else {
		checkIdFormat("VirtualId", state.VirtualId)
		state.Uuid = MakeVirtualNodeId(state.NodeId, state.VirtualId)
	}
	stateJSON, err := json.Marshal(state)
	if nil != err {
		log.Panic("NodeProperties数据序列化错误", err)
	}
	return NewMessageById(state.Uuid, stateJSON, 0)
}

func mqttSendNodeState(client mqtt.Client, state VirtualNodeState) {
	token := client.Publish(
		TopicOfStates(state.NodeId),
		0,
		false,
		createStateMessage(state).Bytes(),
	)
	if token.Wait() && nil != token.Error() {
		log.Error("发送State消息出错", token.Error())
	}
}

func mqttSendNodeProperties(globals *Globals, client mqtt.Client, properties MainNodeProperties) {
	checkIdFormat("NodeType", properties.NodeType)
	if 0 == len(properties.VirtualNodes) {
		log.Panic("缺少虚拟节点数据")
	}
	if "" == properties.HostOS {
		properties.HostOS = runtime.GOOS
	}
	if "" == properties.HostArch {
		properties.HostArch = runtime.GOARCH
	}
	nodeId := properties.NodeId
	// 更新设备列表参数
	for _, vd := range properties.VirtualNodes {
		// 自动生成UUID
		if "" == vd.VirtualId {
			log.Panic("必须指定VirtualNode.VirtualId，并确保其节点范围内的唯一性")
		} else {
			if "" != vd.Uuid {
				log.Debugf("虚拟节点[%s]使用自定义Uuid：%s", vd.Description, vd.Uuid)
			} else {
				checkIdFormat("VirtualId", vd.VirtualId)
				vd.Uuid = MakeVirtualNodeId(nodeId, vd.VirtualId)
			}
		}
	}
	propertiesJSON, err := json.Marshal(properties)
	if nil != err {
		log.Panic("NodeProperties数据序列化错误", err)
	} else if globals.LogVerbose {
		log.Debug("NodeProperties: " + string(propertiesJSON))
	}
	token := client.Publish(
		TopicOfProperties(nodeId),
		0,
		false,
		NewMessageWith(nodeId, nodeId, propertiesJSON, 0).Bytes(),
	)
	if token.Wait() && nil != token.Error() {
		log.Error("发送Properties消息出错", token.Error())
	}
}

func scheduleSendProperties(shutdown context.Context, inspectTask func()) {
	// 在1分钟内上报Properties消息
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	tick := 1
	for {
		select {
		case <-ticker.C:
			inspectTask()
			tick++
			if tick >= 6 {
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
			break
		}
	}
}
