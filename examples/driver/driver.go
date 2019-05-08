package main

import (
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

		driver.Process(func(evt edgex.Packet) {
			ts, _ := strconv.ParseInt(string(evt.Bytes()), 10, 64)
			diff := time.Now().UnixNano() - ts

			ctx.Log().Debug("收到数据用时: ", time.Duration(diff))
			//echo, err := driver.Execute("EXAMPLE-PINGPONG",
			//	edgex.PacketOfString("HELLO"), time.Second)
			//if nil != err {
			//	ctx.Log().Error("发生错误", err)
			//}
			//ctx.Log().Debug("ECHO: ", string(echo.Bytes()))
		})

		ctx.Log().Debugf("创建Driver节点: [%s]", opts.Name)

		driver.Startup()
		defer driver.Shutdown()

		return ctx.AwaitTerm()
	})
}
