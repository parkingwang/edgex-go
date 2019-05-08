package edgex

import (
	"fmt"
	"strings"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

const (
	topicTrigger             = "$EDGEX/EVENTS/${user-topic}"
	topicEndpointRequestQ    = "$EDGEX/EP-REQ/${epid}"
	topicEndpointReplyPrefix = "$EDGEX/EP-REP/"
	topicEndpointReplyQ      = topicEndpointReplyPrefix + "${epid}"
)

func topicOfTrigger(topic string) string {
	if strings.HasPrefix(topic, "/") {
		log.Panicf("Topic MUST NOT starts with '/', was: %s", topic)
	}
	return topicFormat(topicTrigger, "${user-topic}", topic)
}

func topicOfWill(typeName, name string) string {
	return fmt.Sprintf("$EDGEX/WILL/%s/%s", typeName, name)
}

func topicOfEndpointRequestQ(endpointId string) string {
	return topicFormat(topicEndpointRequestQ, "${epid}", endpointId)
}

func topicOfEndpointReplyQ(endpointId string) string {
	return topicFormat(topicEndpointReplyQ, "${epid}", endpointId)
}

func endpointIdOfReplyQ(topic string) string {
	return topic[len(topicEndpointReplyPrefix):]
}

func topicFormat(tpl, key, value string) string {
	return strings.Replace(tpl, key, value, 1)
}
