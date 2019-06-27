package edgex

import "time"

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// 全局配置
type Globals struct {
	MqttBroker            string
	MqttUsername          string
	MqttPassword          string
	MqttQoS               uint8
	MqttRetained          bool
	MqttKeepAlive         time.Duration
	MqttPingTimeout       time.Duration
	MqttConnectTimeout    time.Duration
	MqttReconnectInterval time.Duration
	MqttAutoReconnect     bool
	MqttCleanSession      bool
	MqttMaxRetry          int
	MqttQuitMillSec       uint
}
