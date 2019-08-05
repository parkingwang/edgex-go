package edgex

import (
	"context"
	"encoding/json"
	"github.com/eclipse/paho.mqtt.golang"
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
}

////

func mqttSendInspectMessage(client mqtt.Client, nodeId string, node MainNodeInfo) {
	checkIdFormat("NodeType", node.NodeType)
	if 0 == len(node.VirtualNodes) {
		log.Panic("缺少虚拟节点数据")
	}
	// 自动更新虚拟设备的参数
	if "" != node.NodeId {
		log.Debugf("MainNodeInfo.NodeId 已设置为<%s>，它将被自动覆盖更新", node.NodeId)
	}
	node.NodeId = nodeId
	if "" == node.HostOS {
		node.HostOS = runtime.GOOS
	}
	if "" == node.HostArch {
		node.HostArch = runtime.GOARCH
	}
	// 更新设备列表参数
	for _, vd := range node.VirtualNodes {
		// 自动生成UUID
		if "" == vd.VirtualId {
			log.Panic("必须指定VirtualNode.VirtualId，并确保其节点范围内的唯一性")
		} else {
			checkIdFormat("VirtualId", vd.VirtualId)
			if "" != vd.Uuid {
				log.Debugf("VirtualNodeInfo.Uuid 已设置为<%s>，它将被自动覆盖更新", vd.Uuid)
			}
			vd.Uuid = MakeVirtualNodeId(nodeId, vd.VirtualId)
		}
	}
	data, err := json.Marshal(node)
	if nil != err {
		log.Panic("Inspect数据序列化错误", err)
	}
	// 发送Inspect消息，其中消息来源为NodeId
	token := client.Publish(
		TopicSubscribeNodesInspect,
		0,
		false,
		NewMessageWith(nodeId, nodeId, data, 0).Bytes(),
	)
	if token.Wait() && nil != token.Error() {
		log.Error("发送Inspect消息出错", token.Error())
	}
}

func scheduleSendInspect(shutdown context.Context, inspectTask func()) {
	// 在1分钟内上报Inspect消息
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
