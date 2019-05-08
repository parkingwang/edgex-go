package main

import (
	"fmt"
	"github.com/yoojia/edgex"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

func main() {
	edgex.Run(func(ctx edgex.Context) error {
		//config := ctx.LoadConfig()
		// 向系统注册节点
		opts := edgex.EndpointOptions{
			Id:    "EXAMPLE-TIMER",
			Topic: "scheduled/timer",
		}
		endpoint := ctx.NewEndpoint(opts)

		endpoint.Startup()
		defer endpoint.Shutdown()

		ctx.Log().Debugf("创建Endpoint节点: [%s]", opts.Id)

		// 模拟定时发送消息到系统
		timer := time.NewTicker(time.Second * 3)
		defer timer.Stop()

		for {
			select {
			case t := <-timer.C:
				msg := []byte(fmt.Sprintf("HELLO: %d", t.Unix()))
				// Send
				if err := endpoint.Send(msg); nil != err {
					return err
				}

			case msg := <-endpoint.RecvChan():
				ctx.Log().Debugf("Recv: ", msg)

			case <-ctx.WaitChan():
				return nil

			}
		}

	})
}
