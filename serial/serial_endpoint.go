package main

import (
	"github.com/tarm/serial"
	"github.com/yoojia/edgex"
	"github.com/yoojia/go-value"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
// 使用串口接收POST方式发送的原始数据，并推送到MQTT服务器。

func main() {
	edgex.Run(func(ctx edgex.Context) error {
		config := ctx.LoadConfig()
		name := value.Of(config["Name"]).String()
		rpcAddr := value.Of(config["RpcAddress"]).String()
		broadcast := value.Of(config["Broadcast"]).BoolOrDefault(false)

		endpoint := ctx.NewEndpoint(edgex.EndpointOptions{
			Name:    name,
			RpcAddr: rpcAddr,
		})

		rawOpts := value.Of(config["SerialOptions"]).MustMap()
		serialOpts := getSerialConfig(rawOpts, time.Second*1)
		port, err := serial.OpenPort(serialOpts)
		if nil != err {
			ctx.Log().Panic("打开串口出错: ", err)
			return err
		}

		ctx.Log().Debug("开启串口: ", serialOpts.Name)
		defer func() {
			ctx.Log().Debug("关闭串口: ", serialOpts.Name)
			port.Close()
		}()

		endpoint.Startup()
		defer endpoint.Shutdown()

		buffer := make([]byte, value.Of(rawOpts["bufferSize"]).Int64OrDefault(1024))
		endpoint.Serve(func(in edgex.Message) (out edgex.Message) {
			if n, err := port.Write(in.Bytes()); nil != err {
				ctx.Log().Error("串口写数据出错: ", err)
				return edgex.NewMessageString("ERR:" + err.Error())
			} else if n != in.Size() {
				return edgex.NewMessageString("EX=ERR:WRITE_NOT_FINISHED")
			}
			if broadcast {
				return edgex.NewMessageString("EX=OK:BROADCAST")
			}
			// Read
			if n, err := port.Read(buffer); nil != err {
				return edgex.NewMessageString("EX=ERR:" + err.Error())
			} else {
				return edgex.NewMessageBytes(buffer[:n])
			}
		})

		return ctx.TermAwait()
	})
}
