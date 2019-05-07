package edgex

import (
	"strings"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

const (
	MqTopicEndpointSendQ = "$EdgeX/endpoints/sendq/${UUID}/T/${TOPIC}"
	MqTopicEndpointRecvQ = "$EdgeX/endpoints/recvq/${UUID}/T/${TOPIC}"
	MqTopicPipeline      = "$EdgeX/pipelines/T/${TOPIC}"
	MqTopicDrivers       = "$EdgeX/drivers/T/${TOPIC}"
)

func TopicEndpointSendQ(topic string, uuid string) string {
	return makeEndpoint(MqTopicEndpointSendQ, topic, uuid)
}

func TopicEndpointRecvQ(topic string, uuid string) string {
	return makeEndpoint(MqTopicEndpointRecvQ, topic, uuid)
}

func TopicPipeline(topic string) string {
	checkTopicPrefix(topic)
	return strings.Replace(MqTopicPipeline, "${TOPIC}", topic, 1)
}

func TopicDriver(topic string) string {
	checkTopicPrefix(topic)
	return strings.Replace(MqTopicDrivers, "${TOPIC}", topic, 1)
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
