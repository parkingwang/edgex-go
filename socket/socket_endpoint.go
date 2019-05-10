package main

import (
	"github.com/yoojia/edgex"
	"github.com/yoojia/go-value"
	"strings"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

func main() {
	edgex.Run(func(ctx edgex.Context) error {
		config := ctx.LoadConfig()
		name := value.Of(config["Name"]).String()
		rpcAddress := value.Of(config["RpcAddress"]).String()

		sockOpts := value.Of(config["SocketClientOptions"]).MustMap()
		targetAddress := value.Of(sockOpts["targetAddress"]).String()

		// 向系统注册节点
		opts := edgex.EndpointOptions{
			Name:    name,
			RpcAddr: rpcAddress,
		}
		endpoint := ctx.NewEndpoint(opts)

		ctx.Log().Debugf("连接目标地址: [%s]", targetAddress)
		network := "tcp"
		address := targetAddress
		if strings.Contains(address, "://") {
			network = strings.Split(address, "://")[0]
			address = strings.Split(address, "://")[1]
		}
		client := NewSockClient(SockOptions{
			Network:           network,
			Addr:              address,
			BufferSize:        uint(value.Of(sockOpts["bufferSize"]).Int64OrDefault(1024)),
			ReadTimeout:       value.Of(sockOpts["readTimeout"]).DurationOfDefault(time.Second),
			WriteTimeout:      value.Of(sockOpts["writeTimeout"]).DurationOfDefault(time.Second),
			AutoReconnect:     value.Of(sockOpts["autoReconnect"]).BoolOrDefault(true),
			ReconnectInterval: value.Of(sockOpts["reconnectInterval"]).DurationOfDefault(time.Second * 3),
			KeepAlive:         value.Of(sockOpts["keepAlive"]).BoolOrDefault(true),
			KeepAliveInterval: value.Of(sockOpts["keepAliveInterval"]).DurationOfDefault(time.Second * 3),
		})

		endpoint.Serve(func(in edgex.Message) (out edgex.Message) {
			if ret, err := client.Execute(in.Bytes()); nil != err {
				ctx.Log().Error("Execute failed", err)
				return edgex.NewMessageString("ERR:" + err.Error())
			} else {
				return edgex.NewMessageBytes(ret)
			}
		})

		endpoint.Startup()
		defer endpoint.Shutdown()

		return ctx.AwaitTerm()
	})
}
