package edgex

import (
	"strings"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

const (
	prefixProperties = "edge-x/properties/"
	prefixEvents     = "edge-x/events/"
	prefixValues     = "edge-x/values/"
	prefixStates     = "edge-x/states/"
	prefixActions    = "edge-x/actions/"
	prefixStatistics = "edge-x/statistics/"
	prefixRequests   = "edge-x/requests/"
	prefixReplies    = "edge-x/replies/"
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

func TopicOfActions(nodeId string) string {
	checkTopicAllowed(nodeId)
	return prefixActions + nodeId
}

func TopicOfProperties(nodeId string) string {
	checkTopicAllowed(nodeId)
	return prefixProperties + nodeId
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

func checkTopicAllowed(topic string) {
	if strings.HasPrefix(topic, "/") {
		log.Panicf("Topic MUST NOT starts with '/', was: %s", topic)
	}
}
