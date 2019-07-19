package main

import (
	"fmt"
	"github.com/nextabc-lab/edgex-go"
	"strconv"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

func main() {
	edgex.Run(func(ctx edgex.Context) error {

		opts := edgex.DriverOptions{
			NodeName: "EXAMPLE-DRIVER",
			Topics: []string{
				"example/+",
				"scheduled/+",
				edgex.TopicNodesInspect,
			},
		}

		ctx.Initial(opts.NodeName)
		driver := ctx.NewDriver(opts)

		const testEndpointAddr = "127.0.0.1:5570"

		driver.Process(func(inMsg edgex.Message) {
			recv, _ := strconv.ParseInt(string(inMsg.Body()), 10, 64)

			ctx.Log().Debug("Driver用时: ", time.Duration(time.Now().UnixNano()-recv))
			execStart := time.Now()
			_, err := driver.Execute(testEndpointAddr, inMsg, time.Second*3)
			if nil != err {
				ctx.Log().Error("Execute发生错误: ", err)
			} else {
				ctx.Log().Debug("Execute用时: ", time.Since(execStart))
			}
		})

		ctx.Log().Debugf("创建Driver节点: [%s]", opts.NodeName)

		driver.Startup()
		defer driver.Shutdown()

		timer := time.NewTicker(time.Second)
		for {
			select {
			case <-timer.C:
				execStart := time.Now()
				rep, err := driver.Execute(
					testEndpointAddr,
					driver.NextMessage(opts.NodeName, []byte(fmt.Sprintf("%v", time.Now().UnixNano()))),
					time.Second)
				if nil != err {
					ctx.Log().Error("ScheduleExecute发生错误: ", err)
				} else {
					ctx.Log().Debug("ScheduleExecute用时: ", time.Since(execStart))
					sid := rep.SourceNodeId()
					body := string(rep.Body())
					ctx.Log().Debug("SourceNodeId: " + sid + ", Body: " + body)
				}

			case <-ctx.TermChan():
				timer.Stop()
				return nil
			}
		}

	})
}
