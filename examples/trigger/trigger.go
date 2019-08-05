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
			Topic:           "example/timer",
			AutoInspectFunc: autoNodeFunc(),
		}
		trigger := ctx.NewTrigger(opts)

		trigger.Startup()
		defer trigger.Shutdown()

		ctx.Log().Debugf("创建Trigger节点: [%s]", nodeId)

		timer := time.NewTicker(time.Millisecond * 10)
		defer timer.Stop()

		for {
			select {
			case c := <-timer.C:
				data := fmt.Sprintf("%d", c.UnixNano())
				if e := trigger.PublishEvent("TIMER", []byte(data)); nil != e {
					ctx.Log().Error("Trigger发送消息失败")
				}

			case <-ctx.TermChan():
				return nil
			}
		}

	})
}

func autoNodeFunc() func() edgex.MainNodeInfo {
	return func() edgex.MainNodeInfo {
		return edgex.MainNodeInfo{
			NodeType: edgex.NodeTypeTrigger,
			VirtualNodes: []*edgex.VirtualNodeInfo{
				{
					VirtualId: "TIMER",
					Desc:      "演示Trigger",
				},
			},
		}
	}
}
