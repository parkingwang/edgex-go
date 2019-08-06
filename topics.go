package edgex

import (
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
	TopicSubscribeNodesOffline = prefixNodes + "offline"
	TopicSubscribeNodesEvents  = prefixEvents + "#"
	TopicSubscribeNodesValues  = prefixValues + "#"

	TopicPublishNodesOffline = TopicSubscribeNodesOffline
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

func topicOfRequestSend(executorNodeId, callerNodeId string) string {
	// prefix / ExecutorNodeId / CallerNodeId
	return prefixRequests + executorNodeId + "/" + callerNodeId
}

func topicToRequestCaller(topic string) string {
	// prefix / ExecutorNodeId / CallerNodeId
	idx := strings.LastIndex(topic, "/")
	return topic[idx+1:]
}

func topicOfRequestListen(callerNodeId string) string {
	return prefixRequests + callerNodeId + "/+"
}

func topicOfRepliesSend(executorNodeId, callerNodeId string) string {
	// prefix / CallerNodeId / ExecutorNodeId
	return prefixReplies + callerNodeId + "/" + executorNodeId
}

func topicOfRepliesFilter(executorNodeId, callerNodeId string) string {
	return topicOfRepliesSend(executorNodeId, callerNodeId)
}

func topicOfRepliesListen(callerNodeId string) string {
	return prefixReplies + callerNodeId + "/+"
}

func checkTopic(topic string) {
	if strings.HasPrefix(topic, "/") {
		log.Panicf("Topic MUST NOT starts with '/', was: %s", topic)
	}
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
