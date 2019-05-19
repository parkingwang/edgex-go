package main

import (
	"github.com/nextabc-lab/edgex"
	"github.com/yoojia/go-value"
	lua "github.com/yuin/gopher-lua"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
// 使用Libav命令客户端作为Endpoint，接收GRPC控制指令，并转发到指定Socket客户端

func main() {
	edgex.Run(func(ctx edgex.Context) error {
		config := ctx.LoadConfig()
		name := value.Of(config["Name"]).String()
		rpcAddress := value.Of(config["RpcAddress"]).String()
		scriptFile := value.Of(config["Script"]).String()

		endpoint := ctx.NewEndpoint(edgex.EndpointOptions{
			Name:    name,
			RpcAddr: rpcAddress,
		})

		script := lua.NewState(lua.Options{
			CallStackSize: 8,
			RegistrySize:  8,
		})

		if err := script.DoFile(scriptFile); nil != err {
			ctx.Log().Panic("Lua load script failed: ", err)
		}

		endpoint.Serve(func(in edgex.Message) (out edgex.Message) {
			// 先函数，后参数，正序入栈:
			script.Push(script.GetGlobal("endpointMain"))
			script.Push(lua.LString(string(in.Bytes())))
			if err := script.PCall(1, 2, nil); nil != err {
				return edgex.NewMessageString("EX=ERR:" + err.Error())
			} else {
				retData := script.ToString(1)
				retErr := script.ToString(2)
				script.Pop(2)
				if "" != retErr {
					return edgex.NewMessageString("EX=ERR:" + retErr)
				} else {
					return edgex.NewMessageString(retData)
				}
			}
		})

		endpoint.Startup()
		defer endpoint.Shutdown()

		return ctx.TermAwait()
	})
}
