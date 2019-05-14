package main

import (
	"github.com/nextabc-lab/edgex"
	utils "github.com/nextabc-lab/edgex/serial"
	"github.com/tarm/serial"
	"github.com/yoojia/go-value"
	"os"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
// 使用串口接收POST方式发送的原始数据，并推送到MQTT服务器。

func main() {
	edgex.Run(func(ctx edgex.Context) error {
		config := ctx.LoadConfig()
		name := value.Of(config["Name"]).String()
		topic := value.Of(config["Topic"]).String()

		trigger := ctx.NewTrigger(edgex.TriggerOptions{
			Name:  name,
			Topic: topic,
		})

		rawOpts := value.Of(config["SerialPortOptions"]).MustMap()
		serialOpts := utils.GetSerialConfig(rawOpts, time.Second*1)
		port, err := serial.OpenPort(serialOpts)
		if nil != err {
			ctx.Log().Panic("打开串口出错: ", err)
		} else {
			ctx.Log().Debug("开启串口: ", serialOpts.Name)
			defer func() {
				ctx.Log().Debug("关闭串口: ", serialOpts.Name)
				port.Close()
			}()
		}

		// 启用Trigger服务
		trigger.Startup()
		defer trigger.Shutdown()

		buffer := make([]byte, value.Of(rawOpts["bufferSize"]).Int64OrDefault(1024))
		for {
			select {
			case <-ctx.TermChan():
				return port.Close()

			default:
				if n, err := port.Read(buffer); nil != err {
					ctx.Log().Error("串口读取数据出错: ", err)
					if err == os.ErrClosed {
						return nil
					} else if pe, ok := err.(*os.PathError); ok && pe.Timeout() {
						continue
					} else {
						return err
					}
				} else {
					if err := trigger.Triggered(edgex.NewMessageBytes(buffer[:n])); nil != err {
						ctx.Log().Error("Trigger发送出错: ", err)
					}
				}
			}
		}
	})
}
