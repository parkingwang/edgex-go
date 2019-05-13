package main

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/tidwall/evio"
	"github.com/yoojia/edgex"
	dongk "github.com/yoojia/edgex/dongkong"
	"github.com/yoojia/go-jsonx"
	"github.com/yoojia/go-value"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

func main() {
	edgex.Run(func(ctx edgex.Context) error {
		config := ctx.LoadConfig()
		name := value.Of(config["Name"]).String()
		topic := value.Of(config["Topic"]).String()

		trigger := ctx.NewTrigger(edgex.TriggerOptions{
			Name:  name,
			Topic: topic,
		})

		var server evio.Events

		opts := value.Of(config["SocketServerOptions"]).MustMap()
		server.NumLoops = 1

		server.Data = func(c evio.Conn, in []byte) (out []byte, action evio.Action) {
			cmd, err := dongk.ParseCommand(in)
			if nil != err {
				ctx.Log().Errorw("接收到非东控数据格式数据", "error", err, "data", in)
				return []byte("EX=ERR:INVALID_CMD_SIZE"), action
			}
			// 非监控数据，忽略
			if cmd.FuncId != dongk.DkFunIdBoardState {
				ctx.Log().Debug("接收到非监控状态数据")
				return []byte("EX=ERR:INVALID_STATE"), action
			}
			// 控制指令数据：
			// 0. 无记录
			// 1. 刷卡消息
			// 2. 门磁、按钮、设备启动、远程开门记录
			// 3. 报警记录
			// Reader一个按顺序读取字节数组的包装类
			reader := edgex.WrapByteReader(cmd.Data[:], binary.LittleEndian)
			json := jsonx.NewFatJSON()
			json.Field("sn", cmd.SerialNum)
			json.Field("card", hex.EncodeToString(reader.GetBytesSize(4)))
			reader.GetBytesSize(7) // 丢弃timestamp数据
			json.Field("index", reader.GetUint32())
			json.Field("type", reader.GetByte())
			json.Field("state", reader.GetByte())
			json.Field("doorId", reader.GetByte())
			json.Field("direct", reader.GetByte())
			json.Field("reason", reader.GetByte())
			bytes := json.Bytes()
			// 最后执行控制指令：刷卡数据
			if len(bytes) >= len(`{"A":0}`) {
				if err := trigger.Triggered(edgex.NewMessageBytes(in)); nil != err {
					ctx.Log().Error("触发事件出错: ", err)
					return []byte("EX=ERR:" + err.Error()), action
				} else {
					return []byte("EX=OK:DK_EVENT"), action
				}
			} else {
				ctx.Log().Error("无刷卡事件数据")
				return []byte("EX=ERR:EMPTY_EVENT"), action
			}

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

		address := value.Of(opts["address"]).MustStringArray()
		ctx.Log().Debug("开启Evio服务端: ", address)
		defer ctx.Log().Debug("停止Evio服务端")
		return evio.Serve(server, address...)
	})
}
