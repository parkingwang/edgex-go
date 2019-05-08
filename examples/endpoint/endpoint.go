package main

import (
	"github.com/yoojia/edgex"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

func main() {
	edgex.Run(func(ctx edgex.Context) error {
		//config := ctx.LoadConfig()
		// 向系统注册节点
		opts := edgex.EndpointOptions{
			Id: "EXAMPLE-PINGPONG",
		}
		endpoint := ctx.NewEndpoint(opts)

		endpoint.Serve(func(in edgex.Packet) (out edgex.Packet) {
			ctx.Log().Debug("Recv: ", string(in))
			return in
		})

		endpoint.Startup()
		defer endpoint.Shutdown()

		ctx.Log().Debugf("创建Endpoint节点: [%s]", opts.Id)

		return ctx.AwaitTerm()
	})
}
