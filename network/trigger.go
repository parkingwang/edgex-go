package main

import (
	"github.com/tidwall/evio"
	"github.com/yoojia/edgex"
	"github.com/yoojia/go-value"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//
//

func main() {
	edgex.Run(func(ctx edgex.Context) error {
		config := ctx.LoadConfig()
		name := value.Of(config["Name"]).String()
		topic := value.Of(config["Topic"]).String()
		address := value.Of(config["Address"]).MustStringArray()

		trigger := ctx.NewTrigger(edgex.TriggerOptions{
			Name:  name,
			Topic: topic,
		})

		var server evio.Events
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

		go func() {
			ctx.Log().Debug("开启Evio服务端: ", address)
			if err := evio.Serve(server, address...); err != nil {
				ctx.Log().Error("停止Evio服务端出错: ", err)
			} else {
				ctx.Log().Debug("停止Evio服务端")
			}
		}()

		trigger.Startup()
		defer trigger.Shutdown()

		return ctx.AwaitTerm()
	})
}
