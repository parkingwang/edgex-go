package edgex

import (
	"fmt"
	"strings"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

const (
	topicDevicesInspect = "$EDGEX/DEVICES/INSPECT"
	topicDevicesOffline = "$EDGEX/DEVICES/OFFLINE/%s/%s"
	topicDevicesAlive   = "$EDGEX/DEVICES/ALIVE/%s/%s"
	topicTrigger        = "$EDGEX/EVENTS/${user-topic}"
)

func topicOfTrigger(topic string) string {
	if strings.HasPrefix(topic, "/") {
		log.Panicf("Topic MUST NOT starts with '/', was: %s", topic)
	}
	return topicFormat(topicTrigger, "${user-topic}", topic)
}

func topicOfOffline(typeName, name string) string {
	return fmt.Sprintf(topicDevicesOffline, typeName, name)
}

func topicOfAlive(typeName, name string) string {
	return fmt.Sprintf(topicDevicesAlive, typeName, name)
}

func topicFormat(tpl, key, value string) string {
	return strings.Replace(tpl, key, value, 1)
}
