package edgex

import (
	"fmt"
	"strings"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

const (
	tNodesInspect = "$EdgeX/nodes/inspect"
	tNodesOffline = "$EdgeX/nodes/offline/%s/%s"
	tNodesEvents  = "$EdgeX/events/${user-topic}"
)

const (
	TopicNodesInspect = tNodesInspect
	TopicNodesOffline = "$EdgeX/nodes/offline/#"
	TopicNodesEvents  = "$EdgeX/events/#"
)

func topicOfTriggerEvents(topic string) string {
	if strings.HasPrefix(topic, "/") {
		log.Panicf("Topic MUST NOT starts with '/', was: %s", topic)
	}
	return topicFormat(tNodesEvents, "${user-topic}", topic)
}

func topicOfOffline(typeName, name string) string {
	return fmt.Sprintf(tNodesOffline, typeName, name)
}

func topicFormat(tpl, key, value string) string {
	return strings.Replace(tpl, key, value, 1)
}
