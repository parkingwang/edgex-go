package main

import (
	"github.com/nextabc-lab/edgex-go"
	"runtime"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

func main() {
	edgex.Run(func(ctx edgex.Context) error {
		//config := ctx.LoadConfig()
		// 向系统注册节点
		opts := edgex.EndpointOptions{
			Name:    "EXAMPLE-PINGPONG",
			RpcAddr: "0.0.0.0:6670",
			InspectFunc: func() edgex.Inspect {
				return edgex.Inspect{
					OS:   runtime.GOOS,
					Arch: runtime.GOARCH,
					Devices: []edgex.Device{
						{
							Name:    "ENDPOINT/EXAMPLE-PINGPONG",
							Virtual: false,
							Desc:    "演示终端",
							Command: "ECHO",
						},
					},
				}
			},
		}
		endpoint := ctx.NewEndpoint(opts)

		endpoint.Serve(func(in edgex.Message) (out edgex.Message) {
			name := string(in.Name())
			body := string(in.Body())
			ctx.Log().Debugf("接收到数据, From: %s, Body: %s ", name, body)
			return edgex.NewMessageString("ABCD", "ECHO")
		})

		endpoint.Startup()
		defer endpoint.Shutdown()

		ctx.Log().Debugf("创建Endpoint节点: [%s]", opts.Name)

		return ctx.TermAwait()
	})
}
