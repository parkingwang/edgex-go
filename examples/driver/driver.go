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
				edgex.TopicDeviceInspect,
			},
		}

		driver := ctx.NewDriver(opts)

		const testEndpointAddr = "127.0.0.1:5570"

		driver.Process(func(msg edgex.Message) {
			recv, _ := strconv.ParseInt(string(msg.Body()), 10, 64)

			ctx.Log().Debug("Driver用时: ", time.Duration(time.Now().UnixNano()-recv))
			execStart := time.Now()
			_, err := driver.Execute(testEndpointAddr, msg, time.Second*3)
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
				rep, err := driver.Execute(testEndpointAddr,
					edgex.NewMessageString(opts.NodeName, fmt.Sprintf("%v", time.Now().UnixNano())),
					time.Second)
				if nil != err {
					ctx.Log().Error("ScheduleExecute发生错误: ", err)
				} else {
					ctx.Log().Debug("ScheduleExecute用时: ", time.Since(execStart))
					name := string(rep.Name())
					body := string(rep.Body())
					ctx.Log().Debug("Name: " + name + ", Body: " + body)
				}

			case <-ctx.TermChan():
				timer.Stop()
				return nil
			}
		}

	})
}
