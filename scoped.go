package edgex

import "time"

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// 全局配置
type GlobalScoped struct {
	MqttBroker         string
	MqttQoS            uint8
	MqttRetained       bool
	MqttKeepAlive      time.Duration
	MqttPingTimeout    time.Duration
	MqttConnectTimeout time.Duration
	MqttAutoReconnect  bool
}

func NewDefaultGlobalScoped(broker string) *GlobalScoped {
	return &GlobalScoped{
		MqttBroker:         broker,
		MqttQoS:            1,
		MqttRetained:       false,
		MqttKeepAlive:      time.Second * 3,
		MqttPingTimeout:    time.Second * 1,
		MqttConnectTimeout: time.Second * 5,
		MqttAutoReconnect:  true,
	}
}
