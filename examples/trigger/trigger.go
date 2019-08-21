package main

import (
	"fmt"
	"github.com/nextabc-lab/edgex-go"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

func main() {
	edgex.Run(func(ctx edgex.Context) error {
		// 向系统注册节点
		nodeId := "EXAMPLE-TRIGGER"

		ctx.Initial(nodeId)

		opts := edgex.TriggerOptions{
			Topic:              "example/timer",
			NodePropertiesFunc: makeProperties,
		}
		trigger := ctx.NewTrigger(opts)

		trigger.Startup()
		defer trigger.Shutdown()

		ctx.Log().Debugf("创建Trigger节点: [%s]", nodeId)

		timer := time.NewTicker(time.Second)

		for {
			select {
			case c := <-timer.C:
				data := fmt.Sprintf("%d", c.UnixNano())
				eventId := trigger.GenerateEventId()
				if e := trigger.PublishEvent(
					"TIMER",
					"Major",
					"",
					[]byte(data),
					eventId,
				); nil != e {
					ctx.Log().Error("Trigger发送消息失败")
				} else {
					ctx.Log().Debugf("Trigger发送消息： %d", eventId)
				}

			case <-ctx.TermChan():
				timer.Stop()
				return nil
			}
		}

	})
}

func makeProperties() edgex.MainNodeProperties {
	return edgex.MainNodeProperties{
		NodeType: edgex.NodeTypeTrigger,
		VirtualNodes: []*edgex.VirtualNodeProperties{
			{
				GroupId:     "TIMER",
				MajorId:     "TIMER",
				Description: "演示Trigger",
			},
		},
	}
}
