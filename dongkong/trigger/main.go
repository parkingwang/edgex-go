package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/nextabc-lab/edgex"
	"github.com/nextabc-lab/edgex/dongkong"
	"github.com/tidwall/evio"
	"github.com/yoojia/go-jsonx"
	"github.com/yoojia/go-value"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

const (
	// 设备地址格式：　TRIGGER/SN/DOOR_ID/DIRECT
	deviceAddr = "TRIGGER/%d/%d/%s"
	// 设备数量： 4门进出，共8个输入设备
	deviceCount = 8
)

func main() {
	edgex.Run(func(ctx edgex.Context) error {
		config := ctx.LoadConfig()
		triggerName := value.Of(config["Name"]).String()
		topic := value.Of(config["Topic"]).String()

		boardOpts := value.Of(config["BoardOptions"]).MustMap()
		serialNumber := uint32(value.Of(boardOpts["serialNumber"]).MustInt64())

		trigger := ctx.NewTrigger(edgex.TriggerOptions{
			Name:  triggerName,
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
			if cmd.FuncId != dongk.FunIdBoardState {
				ctx.Log().Debug("接收到非监控状态数据")
				return []byte("EX=ERR:INVALID_STATE"), action
			}
			if cmd.SerialNum != serialNumber {
				ctx.Log().Debug("接收到未知序列号数据")
				return []byte("EX=ERR:UNKNOWN_SN"), action
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
			card := reader.GetBytesSize(4)
			json.Field("card", binary.LittleEndian.Uint32(card))
			json.Field("cardHex", hex.EncodeToString(card))
			reader.GetBytesSize(7) // 丢弃timestamp数据
			json.Field("index", reader.GetUint32())
			json.Field("type", reader.GetByte())
			json.Field("state", reader.GetByte())
			doorId := reader.GetByte()
			direct := reader.GetByte()
			json.Field("doorId", doorId)
			json.Field("direct", dongk.DirectName(direct))
			json.Field("reason", reader.GetByte())
			bytes := json.Bytes()
			// 最后执行控制指令：刷卡数据
			// 地址： TRIGGER/序列号/门号/方向
			deviceName := fmt.Sprintf(deviceAddr, cmd.SerialNum, doorId, dongk.DirectName(direct))
			if err := trigger.Triggered(edgex.NewMessage([]byte(deviceName), bytes)); nil != err {
				ctx.Log().Error("触发事件出错: ", err)
				return []byte("EX=ERR:" + err.Error()), action
			} else {
				return []byte("EX=OK:DK_EVENT"), action
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

		// 上报设备状态
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		go func() {
			doors := make([]edgex.Message, 4)
			for i := 0; i < len(doors); i++ {
				doorId := i + 1
				doors[i] = edgex.NewMessage([]byte(triggerName), makeReaderInfo(serialNumber, doorId, dongk.DirectIn))
				doors[i*2+1] = edgex.NewMessage([]byte(triggerName), makeReaderInfo(serialNumber, doorId, dongk.DirectOut))
			}
			for range ticker.C {
				ctx.Log().Debug("开始上报设备状态消息")
				for _, door := range doors {
					if err := trigger.NotifyAlive(door); nil != err {
						ctx.Log().Error("上报状态消息出错", err)
					}
				}
			}
		}()

		address := value.Of(opts["address"]).MustStringArray()
		ctx.Log().Debug("开启Evio服务端: ", address)
		defer ctx.Log().Debug("停止Evio服务端")
		return evio.Serve(server, address...)
	})
}

func makeReaderInfo(sn uint32, doorId, direct int) []byte {
	directName := dongk.DirectName(byte(direct))
	json := jsonx.NewFatJSON()
	json.Field("name", fmt.Sprintf("TRIGGER/%d/%d/%s", sn, doorId, directName))
	json.Field("desc", fmt.Sprintf("%d号门%s读卡器", doorId, directName))
	json.Field("command", "AT")
	return json.Bytes()
}
