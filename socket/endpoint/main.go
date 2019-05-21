package main

import (
	"github.com/nextabc-lab/edgex"
	"github.com/yoojia/go-value"
	"strings"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
// 使用Socket客户端作为Endpoint，接收GRPC控制指令，并转发到指定Socket客户端

func main() {
	edgex.Run(func(ctx edgex.Context) error {
		config := ctx.LoadConfig()
		name := value.Of(config["Name"]).String()
		rpcAddress := value.Of(config["RpcAddress"]).String()
		// 是否作为广播Endpoint。
		// 如果设置为广播终端，不会接收Socket客户端的响应数据
		broadcast := value.Of(config["Broadcast"]).BoolOrDefault(false)

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
			if broadcast {
				if n, err := client.Send(in.Body()); nil != err {
					return edgex.NewMessageString(name, "EX=ERR:"+err.Error())
				} else if n != in.Size() {
					return edgex.NewMessageString(name, "EX=ERR:WRITE_NOT_FINISHED")
				} else {
					return edgex.NewMessageString(name, "EX=OK:BROADCAST")
				}
			} else {
				if ret, err := client.Execute(in.Body()); nil != err {
					ctx.Log().Error("Execute failed", err)
					return edgex.NewMessageString(name, "EX=ERR:"+err.Error())
				} else {
					return edgex.NewMessage([]byte(name), ret)
				}
			}
		})

		endpoint.Startup()
		defer endpoint.Shutdown()

		return ctx.TermAwait()
	})
}
