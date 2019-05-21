package main

import (
	"github.com/nextabc-lab/edgex"
	"github.com/yoojia/go-value"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
// Echo

func main() {
	edgex.Run(func(ctx edgex.Context) error {
		config := ctx.LoadConfig()
		name := value.Of(config["Name"]).String()
		rpcAddress := value.Of(config["RpcAddress"]).String()

		endpoint := ctx.NewEndpoint(edgex.EndpointOptions{
			Name:    name,
			RpcAddr: rpcAddress,
		})

		endpoint.Serve(func(in edgex.Message) (out edgex.Message) {
			ctx.Log().Debug(">> Receive a message, ECHO")
			return in
		})

		endpoint.Startup()
		defer endpoint.Shutdown()

		return ctx.TermAwait()
	})
}
