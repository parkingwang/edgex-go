package edgex

import (
	"fmt"
	"strings"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

const (
	tDevicesInspect = "$EDGEX/DEVICES/INSPECT"
	tDevicesOffline = "$EDGEX/DEVICES/OFFLINE/%s/%s"
	tDevicesAlive   = "$EDGEX/DEVICES/ALIVE/%s/%s"
	tTrigger        = "$EDGEX/EVENTS/${user-topic}"
)

const (
	TopicDeviceInspect = "$EDGEX/DEVICES/INSPECT/#"
	TopicDeviceOffline = "$EDGEX/DEVICES/OFFLINE/#"
	TopicDeviceALIVE   = "$EDGEX/DEVICES/ALIVE/#"
	TopicDeviceEvents  = "$EDGEX/EVENTS/#"
)

func topicOfTrigger(topic string) string {
	if strings.HasPrefix(topic, "/") {
		log.Panicf("Topic MUST NOT starts with '/', was: %s", topic)
	}
	return topicFormat(tTrigger, "${user-topic}", topic)
}

func topicOfOffline(typeName, name string) string {
	return fmt.Sprintf(tDevicesOffline, typeName, name)
}

func topicOfAlive(typeName, name string) string {
	return fmt.Sprintf(tDevicesAlive, typeName, name)
}

func topicFormat(tpl, key, value string) string {
	return strings.Replace(tpl, key, value, 1)
}
