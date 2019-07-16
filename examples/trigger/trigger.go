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
		opts := edgex.TriggerOptions{
			NodeName:        "EXAMPLE-TRIGGER",
			Topic:           "example/timer",
			InspectNodeFunc: nodeInfo,
		}
		trigger := ctx.NewTrigger(opts)

		trigger.Startup()
		defer trigger.Shutdown()

		ctx.Log().Debugf("创建Trigger节点: [%s]", opts.NodeName)

		timer := time.NewTicker(time.Millisecond * 10)
		defer timer.Stop()

		for {
			select {
			case c := <-timer.C:
				data := fmt.Sprintf("%d", c.UnixNano())
				if e := trigger.SendEventMessage("MAIN-TIMER", []byte(data)); nil != e {
					ctx.Log().Error("Trigger发送消息失败")
				}

			case <-ctx.TermChan():
				return nil
			}
		}

	})
}

func nodeInfo() edgex.EdgeNode {
	return edgex.EdgeNode{
		NodeType: edgex.NodeTypeTrigger,
		VirtualNodes: []edgex.VirtualNode{
			{
				Major:   "TIMER",
				Minor:   "MAIN",
				Virtual: false,
				Desc:    "演示Trigger",
			},
		},
	}
}
