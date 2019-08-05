package main

import (
	"github.com/nextabc-lab/edgex-go"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

func main() {
	edgex.Run(func(ctx edgex.Context) error {
		//config := ctx.LoadConfig()
		// 向系统注册节点
		nodeName := "EXAMPLE-ENDPOINT"

		ctx.Initial(nodeName)

		opts := edgex.EndpointOptions{
			RpcAddr:         "0.0.0.0:5571",
			AutoInspectFunc: autoNodeFunc(),
		}
		endpoint := ctx.NewEndpoint(opts)

		endpoint.Serve(func(in edgex.Message) (out edgex.Message) {
			sid := in.SourceUuid()
			body := string(in.Body())
			ctx.Log().Debugf("接收到数据, SourceUuid: %s, Body: %s ", sid, body)
			return endpoint.NextMessageByVirtualId("ABCD", []byte("ECHO"))
		})

		endpoint.Startup()
		defer endpoint.Shutdown()

		ctx.Log().Debugf("创建Endpoint节点: [%s]", nodeName)

		return ctx.TermAwait()
	})
}

func autoNodeFunc() func() edgex.MainNode {
	return func() edgex.MainNode {
		return edgex.MainNode{
			NodeType: edgex.NodeTypeEndpoint,
			VirtualNodes: []*edgex.VirtualNode{
				{
					VirtualId:  "main",
					Desc:       "演示终端",
					RpcCommand: "ECHO",
				},
			},
		}
	}
}
