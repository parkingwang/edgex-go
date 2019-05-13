package main

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/yoojia/edgex"
	"github.com/yoojia/edgex/vendors/dongkong"
	"github.com/yoojia/go-at"
	"github.com/yoojia/go-value"
	"net"
	"os"
	"strconv"
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

		sockOpts := value.Of(config["SocketClientOptions"]).MustMap()
		remoteAddress := value.Of(sockOpts["remoteAddress"]).String()
		readTimeout := value.Of(sockOpts["readTimeout"]).DurationOfDefault(time.Second)
		writeTimeout := value.Of(sockOpts["writeTimeout"]).DurationOfDefault(time.Second)

		// 向系统注册节点
		opts := edgex.EndpointOptions{
			Name:    name,
			RpcAddr: rpcAddress,
		}
		endpoint := ctx.NewEndpoint(opts)

		ctx.Log().Debugf("连接目标地址: [udp://%s]", remoteAddress)

		addr, err := net.ResolveUDPAddr("udp", remoteAddress)
		if nil != err {
			return errors.WithMessage(err, "Resolve udp address failed")
		}
		udpConn, err := net.DialUDP("udp", nil, addr)
		if nil != err {
			return errors.WithMessage(err, "UDP dial failed")
		}

		boardOpts := value.Of(config["BoardOptions"]).MustMap()
		serialNumber := uint32(value.Of(boardOpts["serialNumber"]).MustInt64())

		parser := at.NewAtRegister()
		parser.Add("OPEN", func(args ...string) (data []byte, err error) {
			switchId, err := strconv.ParseInt(args[0], 10, 64)
			if nil != err {
				return nil, errors.New("INVALID_SWITCH_ID:" + args[0])
			}
			return dongk.NewCommand(dongk.DkFunIdRemoteOpen, serialNumber, 0, [32]byte{byte(switchId)}).Bytes(),
				nil
		})

		buffer := make([]byte, 64)
		endpoint.Serve(func(in edgex.Message) (out edgex.Message) {
			// 转控制指令，转换成DK指令
			dkCmd, err := parser.Apply(string(in.Bytes()))
			if nil != err {
				return edgex.NewMessageString("EX=ERR:" + err.Error())
			}

			cmd, _ := dongk.ParseCommand(dkCmd)

			// Write
			udpConn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if _, err := udpConn.Write(cmd.Bytes()); nil != err {
				return edgex.NewMessageString("EX=ERR:" + err.Error())
			}
			for i := 0; i < 3; i++ {
				udpConn.SetReadDeadline(time.Now().Add(readTimeout))
				if _, err := udpConn.Read(buffer); nil != err {
					if pe, ok := err.(*os.PathError); ok && pe.Timeout() {
						<-time.After(time.Second)
						continue
					} else {
						return edgex.NewMessageString("EX=ERR:" + err.Error())
					}
				}
				// parse
				if retCmd, err := dongk.ParseCommand(buffer); nil != err {
					return edgex.NewMessageString("EX=ERR:" + err.Error())
				} else if retCmd.Data()[0] == 0x01 {
					return edgex.NewMessageString(fmt.Sprintf("EX=OK:%d", cmd.FuncId()))
				} else {
					return edgex.NewMessageString("EX=ERR:" + err.Error())
				}
			}
			return edgex.NewMessageString("EX=ERR:EMPTY_RESPONSE")
		})

		endpoint.Startup()
		defer endpoint.Shutdown()

		return ctx.TermAwait()
	})
}
