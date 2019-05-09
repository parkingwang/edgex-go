package main

import (
	"fmt"
	"github.com/yoojia/edgex"
	"strconv"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

func main() {
	edgex.Run(func(ctx edgex.Context) error {

		opts := edgex.DriverOptions{
			Name: "EXAMPLE-DRIVER",
			Topics: []string{
				"example/+",
				"scheduled/+",
			},
		}

		driver := ctx.NewDriver(opts)

		const testEndpointAddr = "127.0.0.1:5570"

		driver.Process(func(evt edgex.Packet) {
			recv, _ := strconv.ParseInt(string(evt.Bytes()), 10, 64)

			ctx.Log().Debug("Driver用时: ", time.Duration(time.Now().UnixNano()-recv))
			execStart := time.Now()
			_, err := driver.Execute(testEndpointAddr, evt, time.Second*3)
			if nil != err {
				ctx.Log().Error("Execute发生错误: ", err)
			} else {
				ctx.Log().Debug("Execute用时: ", time.Since(execStart))
			}
		})

		ctx.Log().Debugf("创建Driver节点: [%s]", opts.Name)

		driver.Startup()
		defer driver.Shutdown()

		timer := time.NewTicker(time.Second)
		for {
			select {
			case <-timer.C:
				execStart := time.Now()
				_, err := driver.Execute(testEndpointAddr,
					edgex.PacketOfString(fmt.Sprintf("%v", time.Now().UnixNano())),
					time.Second)
				if nil != err {
					ctx.Log().Error("ScheduleExecute发生错误: ", err)
				} else {
					ctx.Log().Debug("ScheduleExecute用时: ", time.Since(execStart))
				}

			case <-ctx.WaitChan():
				timer.Stop()
				return nil
			}
		}

	})
}
