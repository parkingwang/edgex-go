package main

import (
	"encoding/hex"
	"fmt"
	"github.com/nextabc-lab/edgex"
	"github.com/nextabc-lab/edgex/dongkong"
	"github.com/pkg/errors"
	"github.com/yoojia/go-at"
	"github.com/yoojia/go-value"
	"net"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
// 使用Socket客户端作为Endpoint，接收GRPC控制指令，并转发到指定Socket客户端

func main() {
	edgex.Run(func(ctx edgex.Context) error {
		config := ctx.LoadConfig()
		deviceName := value.Of(config["Name"]).String()
		rpcAddress := value.Of(config["RpcAddress"]).String()

		sockOpts := value.Of(config["SocketClientOptions"]).MustMap()
		remoteAddress := value.Of(sockOpts["remoteAddress"]).String()
		readTimeout := value.Of(sockOpts["readTimeout"]).DurationOfDefault(time.Second)
		writeTimeout := value.Of(sockOpts["writeTimeout"]).DurationOfDefault(time.Second)

		boardOpts := value.Of(config["BoardOptions"]).MustMap()
		serialNumber := uint32(value.Of(boardOpts["serialNumber"]).MustInt64())

		// AT指令解析
		atRegistry := at.NewAtRegister()
		atCommands(atRegistry, serialNumber)

		ctx.Log().Debugf("连接目标地址: [udp://%s]", remoteAddress)
		conn, err := makeUdpConn(remoteAddress)
		if nil != err {
			return err
		}

		buffer := make([]byte, 64)
		endpoint := ctx.NewEndpoint(edgex.EndpointOptions{
			Name:    deviceName,
			RpcAddr: rpcAddress,
		})
		endpoint.Serve(func(msg edgex.Message) (out edgex.Message) {
			atCmd := string(msg.Body())
			ctx.Log().Debug("接收到控制指令: " + atCmd)
			cmd, err := atRegistry.Apply(atCmd)
			if nil != err {
				return edgex.NewMessageString(deviceName, "EX=ERR:"+err.Error())
			}
			ctx.Log().Debug("东控指令码: " + hex.EncodeToString(cmd))
			// Write
			if err := tryWrite(conn, cmd, writeTimeout); nil != err {
				return edgex.NewMessageString(deviceName, "EX=ERR:"+err.Error())
			}
			// Read
			var n = int(0)
			for i := 0; i < 2; i++ {
				if n, err = tryRead(conn, buffer, readTimeout); nil != err {
					ctx.Log().Errorf("读取设备响应数据出错[%d]: %s", i, err.Error())
					<-time.After(time.Millisecond * 500)
				} else {
					break
				}
			}
			// parse
			if n > 0 {
				if outCmd, err := dongk.ParseCommand(buffer); nil != err {
					return edgex.NewMessageString(deviceName, "EX=ERR:"+err.Error())
				} else if outCmd.Success() {
					return edgex.NewMessageString(deviceName, fmt.Sprintf("EX=OK"))
				} else {
					return edgex.NewMessageString(deviceName, "EX=ERR:NOT_OK")
				}
			} else {
				return edgex.NewMessageString(deviceName, "EX=ERR:NO_REPLY")
			}

		})

		endpoint.Startup()
		defer endpoint.Shutdown()

		return ctx.TermAwait()
	})
}

func makeUdpConn(remoteAddr string) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if nil != err {
		return nil, errors.WithMessage(err, "Resolve udp address failed")
	}
	return net.DialUDP("udp", nil, addr)
}

func tryWrite(conn *net.UDPConn, bs []byte, to time.Duration) error {
	if err := conn.SetWriteDeadline(time.Now().Add(to)); nil != err {
		return err
	}
	if _, err := conn.Write(bs); nil != err {
		return err
	} else {
		return nil
	}
}

func tryRead(conn *net.UDPConn, buffer []byte, to time.Duration) (n int, err error) {
	if err := conn.SetReadDeadline(time.Now().Add(to)); nil != err {
		return 0, err
	}
	return conn.Read(buffer)
}
