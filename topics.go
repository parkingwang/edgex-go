package edgex

import (
	"strings"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

const (
	tplEndpointSendQ = "$EdgeX/Endpoint/SendQ/${UUID}/$T/${TOPIC}"
	tplEndpointRecvQ = "$EdgeX/Endpoint/RecvQ/${UUID}/$T/${TOPIC}"
	tplPipeline      = "$EdgeX/Pipeline/$T/${TOPIC}"
	tplDrivers       = "$EdgeX/Drivers/$T/${TOPIC}"
)

func topicOfEndpointSendQ(topic string, uuid string) string {
	return makeEndpoint(tplEndpointSendQ, topic, uuid)
}

func topicOfEndpointRecvQ(topic string, uuid string) string {
	return makeEndpoint(tplEndpointRecvQ, topic, uuid)
}

func topicOfPipeline(topic string) string {
	checkTopicPrefix(topic)
	return strings.Replace(tplPipeline, "${TOPIC}", topic, 1)
}

func topicOfDriver(topic string) string {
	checkTopicPrefix(topic)
	return strings.Replace(tplDrivers, "${TOPIC}", topic, 1)
}

func makeEndpoint(tpl, topic, uuid string) string {
	checkTopicPrefix(topic)
	return strings.Replace(
		strings.Replace(tpl, "${TOPIC}", topic, 1),
		"${UUID}", uuid, 1)
}

func checkTopicPrefix(topic string) {
	if strings.HasPrefix(topic, "/") {
		log.Panicf("Topic MUST NOT starts with '/', was: %s", topic)
	}
}
