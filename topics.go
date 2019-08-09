package edgex

import (
	"strings"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

const (
	prefixProperties = "$EdgeX/properties/"
	prefixEvents     = "$EdgeX/events/"
	prefixValues     = "$EdgeX/values/"
	prefixStates     = "$EdgeX/states/"
	prefixStatistics = "$EdgeX/statistics/"
	prefixRequests   = "$EdgeX/requests/"
	prefixReplies    = "$EdgeX/replies/"

	TopicSubscribeProperties = prefixProperties + "#"
	TopicSubscribeEvents     = prefixEvents + "#"
	TopicSubscribeValues     = prefixValues + "#"
	TopicSubscribeStatistics = prefixStatistics + "#"
	TopicSubscribeStates     = prefixStates + "#"
)

func TopicOfEvents(exTopic string) string {
	checkTopicAllowed(exTopic)
	return prefixEvents + exTopic
}

func TopicOfValues(exTopic string) string {
	checkTopicAllowed(exTopic)
	return prefixValues + exTopic
}

func TopicOfStates(nodeId string) string {
	checkTopicAllowed(nodeId)
	return prefixStates + nodeId
}

func TopicOfStatistics(nodeId string) string {
	checkTopicAllowed(nodeId)
	return prefixStatistics + nodeId
}

func TopicOfProperties(nodeId string) string {
	checkTopicAllowed(nodeId)
	return prefixProperties + nodeId
}

func topicOfRequestSend(executorNodeId, callerNodeId string) string {
	// prefix / ExecutorNodeId / CallerNodeId
	return prefixRequests + executorNodeId + "/" + callerNodeId
}

func topicToRequestCaller(exTopic string) string {
	// prefix / ExecutorNodeId / CallerNodeId
	idx := strings.LastIndex(exTopic, "/")
	return exTopic[idx+1:]
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

func checkTopicAllowed(topic string) {
	if strings.HasPrefix(topic, "/") {
		log.Panicf("Topic MUST NOT starts with '/', was: %s", topic)
	}
}

func unwrapEdgeXTopic(mqttRawTopic string) string {
	if "" != mqttRawTopic {
		if strings.HasPrefix(mqttRawTopic, prefixEvents) {
			return mqttRawTopic[len(prefixEvents):]
		} else if strings.HasPrefix(mqttRawTopic, prefixStatistics) {
			return mqttRawTopic[len(prefixStatistics):]
		} else if strings.HasPrefix(mqttRawTopic, prefixValues) {
			return mqttRawTopic[len(prefixValues):]
		} else if strings.HasPrefix(mqttRawTopic, prefixProperties) {
			return mqttRawTopic[len(prefixProperties):]
		} else if strings.HasPrefix(mqttRawTopic, prefixStates) {
			return mqttRawTopic[len(prefixStates):]
		} else {
			return mqttRawTopic
		}
	} else {
		return mqttRawTopic
	}
}
