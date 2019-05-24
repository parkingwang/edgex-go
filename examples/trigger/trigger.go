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
			Name:  "EXAMPLE-TIMER",
			Topic: "example/timer",
		}
		trigger := ctx.NewTrigger(opts)

		trigger.Startup()
		defer trigger.Shutdown()

		ctx.Log().Debugf("创建Trigger节点: [%s]", opts.Name)

		timer := time.NewTicker(time.Millisecond * 10)

		for {
			select {
			case c := <-timer.C:
				pkg := edgex.NewMessageString(opts.Name, fmt.Sprintf("%d", c.UnixNano()))
				if e := trigger.Triggered(pkg); nil != e {
					ctx.Log().Error("Trigger发送消息失败")
				}

			case <-ctx.TermChan():
				timer.Stop()
				return nil
			}
		}

	})
}
