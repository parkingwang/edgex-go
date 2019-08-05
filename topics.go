package edgex

import (
	"fmt"
	"strings"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

const (
	prefixNodes    = "$EdgeX/nodes/"
	prefixEvents   = "$EdgeX/events/"
	prefixValues   = "$EdgeX/values/"
	prefixStats    = "$EdgeX/stats/"
	prefixRequests = "$EdgeX/requests/"
	prefixReplies  = "$EdgeX/replies/"

	TopicSubscribeNodesInspect = prefixNodes + "inspect"
	TopicSubscribeNodesOffline = prefixNodes + "offline/#"
	TopicSubscribeNodesEvents  = prefixEvents + "#"
	TopicSubscribeNodesValues  = prefixValues + "#"
)

func topicOfEvents(topic string) string {
	checkTopic(topic)
	return prefixEvents + topic
}

func topicOfValues(topic string) string {
	checkTopic(topic)
	return prefixValues + topic
}

func topicOfStats(topic string) string {
	checkTopic(topic)
	return prefixStats + topic
}

func topicOfRequestSend(executorNodeId string, seqId uint32, callerNodeId string) string {
	return fmt.Sprintf(prefixRequests+"%s/%d/%s", executorNodeId, seqId, callerNodeId)
}

func topicOfRequestListen(nodeId string) string {
	return fmt.Sprintf(prefixRequests+"%s/+/+", nodeId)
}

func topicOfRepliesSend(executorNodeId string, seqId uint32, callerNodeId string) string {
	return fmt.Sprintf(prefixReplies+"%s/%d/%s", callerNodeId, seqId, executorNodeId)
}

func topicOfRepliesFilter(executorNodeId string, seqId uint32, callerNodeId string) string {
	return topicOfRepliesSend(executorNodeId, seqId, callerNodeId)
}

func topicOfRepliesListen(callerNodeId string) string {
	return fmt.Sprintf(prefixReplies+"%s/+/+", callerNodeId)
}

func checkTopic(topic string) {
	if strings.HasPrefix(topic, "/") {
		log.Panicf("Topic MUST NOT starts with '/', was: %s", topic)
	}
}

func topicOfOffline(typeName, name string) string {
	return fmt.Sprintf(prefixNodes+"offline/%s/%s", typeName, name)
}

func unwrapEdgeXTopic(mqttRawTopic string) string {
	if "" != mqttRawTopic || strings.HasPrefix(mqttRawTopic, "$EdgeX/") {
		if strings.HasPrefix(mqttRawTopic, prefixEvents) {
			return mqttRawTopic[len(prefixEvents):]
		} else if strings.HasPrefix(mqttRawTopic, prefixStats) {
			return mqttRawTopic[len(prefixStats):]
		} else if strings.HasPrefix(mqttRawTopic, prefixValues) {
			return mqttRawTopic[len(prefixValues):]
		} else if strings.HasPrefix(mqttRawTopic, prefixNodes) {
			return mqttRawTopic[len(prefixNodes):]
		} else {
			return mqttRawTopic
		}
	} else {
		return mqttRawTopic
	}
}
