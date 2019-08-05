package edgex

import (
	"fmt"
	"strings"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

const (
	TopicNodesInspect = "$EdgeX/nodes/inspect"
	TopicNodesOffline = "$EdgeX/nodes/offline/#"
	TopicNodesEvents  = "$EdgeX/events/#"
	TopicNodesValues  = "$EdgeX/values/#"
)

func topicOfEvents(topic string) string {
	checkTopic(topic)
	return fmt.Sprintf("$EdgeX/events/%s", topic)
}

func topicOfValues(topic string) string {
	checkTopic(topic)
	return fmt.Sprintf("$EdgeX/values/%s", topic)
}

func topicOfRequestSend(executorNodeId string, seqId uint32, callerNodeId string) string {
	return fmt.Sprintf("$EdgeX/requests/%s/%d/%s", executorNodeId, seqId, callerNodeId)
}

func topicOfRequestListen(nodeId string) string {
	return fmt.Sprintf("$EdgeX/requests/%s/+/+", nodeId)
}

func topicOfRepliesSend(executorNodeId string, seqId uint32, callerNodeId string) string {
	return fmt.Sprintf("$EdgeX/replies/%s/%d/%s", callerNodeId, seqId, executorNodeId)
}

func topicOfRepliesFilter(executorNodeId string, seqId uint32, callerNodeId string) string {
	return topicOfRepliesSend(executorNodeId, seqId, callerNodeId)
}

func topicOfRepliesListen(callerNodeId string) string {
	return fmt.Sprintf("$EdgeX/replies/%s/+/+", callerNodeId)
}

func checkTopic(topic string) {
	if strings.HasPrefix(topic, "/") {
		log.Panicf("Topic MUST NOT starts with '/', was: %s", topic)
	}
}

func topicOfOffline(typeName, name string) string {
	return fmt.Sprintf("$EdgeX/nodes/offline/%s/%s", typeName, name)
}
