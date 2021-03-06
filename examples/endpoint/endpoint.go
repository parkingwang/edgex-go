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
		//config := ctx.LoadConfig()
		// 向系统注册节点
		nodeId := "DEV-ENDPOINT"

		ctx.Initial(nodeId)

		opts := edgex.EndpointOptions{
			NodePropertiesFunc: makeNodeProperties,
		}
		endpoint := ctx.NewEndpoint(opts)

		endpoint.Serve(func(in edgex.Message) (out []byte) {
			ctx.Log().Debugf("Received: " + string(in.Body()))
			return []byte(fmt.Sprintf("TIME: %s", time.Now()))
		})

		endpoint.Startup()
		defer endpoint.Shutdown()

		ctx.Log().Debugf("创建Endpoint节点: [%s]", nodeId)

		return ctx.TermAwait()
	})
}

func makeNodeProperties() edgex.MainNodeProperties {
	return edgex.MainNodeProperties{
		NodeType: edgex.NodeTypeEndpoint,
		VirtualNodes: []*edgex.VirtualNodeProperties{
			{
				BoardId:     "main",
				MajorId:     "main",
				Description: "演示终端",
				StateCommands: map[string]string{
					"echo": "AT+ECHO",
				},
			},
		},
	}
}
