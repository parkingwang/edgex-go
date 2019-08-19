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
	if "" != state.UnionId {
		log.Debugf("NodeState: 虚拟节点使用自定义Uuid：%s", state.UnionId)
	} else {
		state.UnionId = MakeUnionId(state.NodeId, state.GroupId, state.MajorId, state.MinorId)
	}
	stateJSON, err := json.Marshal(state)
	if nil != err {
		log.Panic("NodeState: 数据序列化错误", err)
	}
	return NewMessageByUnionId(state.UnionId, stateJSON, 0)
}

func mqttSendNodeState(client mqtt.Client, state VirtualNodeState) {
	token := client.Publish(
		TopicOfStates(state.NodeId),
		0,
		false,
		createStateMessage(state).Bytes(),
	)
	if token.Wait() && nil != token.Error() {
		log.Error("NodeState: 发送消息出错", token.Error())
	}
}

func mqttSendNodeProperties(globals *Globals, client mqtt.Client, properties MainNodeProperties) {
	checkRequired(properties.NodeType, "NodeType是必须的")
	if 0 == len(properties.VirtualNodes) {
		log.Panic("NodeProperties: 缺少虚拟节点数据")
	}
	if "" == properties.HostOS {
		properties.HostOS = runtime.GOOS
	}
	if "" == properties.HostArch {
		properties.HostArch = runtime.GOARCH
	}
	nodeId := properties.NodeId
	// 更新设备列表参数
	for _, vn := range properties.VirtualNodes {
		if "" != vn.UnionId {
			log.Debugf("NodeProperties: 虚拟节点[%s]使用自定义UnionId：%s", vn.Description, vn.UnionId)
		} else {
			vn.UnionId = MakeUnionId(nodeId, vn.GroupId, vn.MajorId, vn.MinorId)
		}
	}
	propertiesJSON, err := json.Marshal(properties)
	if nil != err {
		log.Panic("NodeProperties: 数据序列化错误", err)
	} else if globals.LogVerbose {
		log.Debug("NodeProperties: " + string(propertiesJSON))
	}
	token := client.Publish(
		TopicOfProperties(nodeId),
		0,
		false,
		NewMessage(nodeId, nodeId, nodeId, "", propertiesJSON, 0).Bytes(),
	)
	if token.Wait() && nil != token.Error() {
		log.Error("NodeProperties: 发送消息出错", token.Error())
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
