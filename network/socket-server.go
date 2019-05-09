package main

import (
	"github.com/tidwall/evio"
	"github.com/yoojia/edgex"
	"github.com/yoojia/go-value"
	"strings"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//
// 使用Evio库作为Socket服务端，支持 tcp, udp, unix 等通讯方式。
// 服务端接收客户端推送的数据包，并以配置的Topic，将消息发送到MQTT服务器。

func main() {
	edgex.Run(func(ctx edgex.Context) error {
		config := ctx.LoadConfig()
		name := value.Of(config["Name"]).String()
		topic := value.Of(config["Topic"]).String()
		address := value.Of(config["SocketAddress"]).MustStringArray()

		trigger := ctx.NewTrigger(edgex.TriggerOptions{
			Name:  name,
			Topic: topic,
		})

		var server evio.Events

		opts := value.Of(config["SocketServerOptions"]).MustMap()
		server.NumLoops = int(value.Of(opts["numLoops"]).Int64OrDefault(1))
		if server.NumLoops > 1 {
			lb := strings.ToLower(value.Of(opts["loadBalance"]).String())
			switch lb {
			case "random":
				server.LoadBalance = evio.Random
			case "roundrobin":
				server.LoadBalance = evio.RoundRobin
			case "leastconnections":
				server.LoadBalance = evio.LeastConnections
			default:
				server.LoadBalance = evio.Random
			}
		}

		server.Data = func(c evio.Conn, in []byte) (out []byte, action evio.Action) {
			// 接收数据，并触发事件
			if err := trigger.Triggered(edgex.PacketOfBytes(in)); nil != err {
				ctx.Log().Error("触发事件出错: ", err)
			}
			return []byte("OK\r\n"), action
		}

		server.Opened = func(c evio.Conn) (out []byte, opts evio.Options, action evio.Action) {
			ctx.Log().Debug("接受客户端: ", c.RemoteAddr())
			return
		}

		server.Closed = func(c evio.Conn, err error) (action evio.Action) {
			ctx.Log().Debug("断开客户端: ", c.RemoteAddr())
			return
		}

		// 启用Trigger服务
		trigger.Startup()
		defer trigger.Shutdown()

		ctx.Log().Debug("开启Evio服务端: ", address)
		defer ctx.Log().Debug("停止Evio服务端")
		return evio.Serve(server, address...)
	})
}
