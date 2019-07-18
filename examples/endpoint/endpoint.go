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
		opts := edgex.EndpointOptions{
			NodeName:        nodeName,
			RpcAddr:         "0.0.0.0:5570",
			InspectNodeFunc: mainNodeFunc(nodeName),
		}
		endpoint := ctx.NewEndpoint(opts)

		endpoint.Serve(func(in edgex.Message) (out edgex.Message) {
			sid := in.SourceNodeId()
			body := string(in.Body())
			ctx.Log().Debugf("接收到数据, SourceNodeId: %s, Body: %s ", sid, body)
			return endpoint.NextMessage("ABCD", []byte("ECHO"))
		})

		endpoint.Startup()
		defer endpoint.Shutdown()

		ctx.Log().Debugf("创建Endpoint节点: [%s]", opts.NodeName)

		return ctx.TermAwait()
	})
}

func mainNodeFunc(nodeName string) func() edgex.MainNode {
	return func() edgex.MainNode {
		return edgex.MainNode{
			NodeType: edgex.NodeTypeEndpoint,
			NodeName: nodeName,
			VirtualNodes: []edgex.VirtualNode{
				{
					NodeId:     "main",
					Desc:       "演示终端",
					RpcCommand: "ECHO",
				},
			},
		}
	}
}
