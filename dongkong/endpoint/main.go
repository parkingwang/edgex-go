package main

import (
	"fmt"
	"github.com/nextabc-lab/edgex"
	"github.com/nextabc-lab/edgex/dongkong"
	"github.com/pkg/errors"
	"github.com/yoojia/go-at"
	"github.com/yoojia/go-value"
	"net"
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

		boardOpts := value.Of(config["BoardOptions"]).MustMap()
		serialNumber := uint32(value.Of(boardOpts["serialNumber"]).MustInt64())

		// AT指令解析
		registry := at.NewAtRegister()
		registryAt(registry, serialNumber)

		ctx.Log().Debugf("连接目标地址: [udp://%s]", remoteAddress)
		conn, err := makeUdpConn(remoteAddress)
		if nil != err {
			return err
		}

		buffer := make([]byte, 64)
		endpoint := ctx.NewEndpoint(edgex.EndpointOptions{
			Name:    name,
			RpcAddr: rpcAddress,
		})
		endpoint.Serve(func(in edgex.Message) (out edgex.Message) {
			cmd, err := registry.Apply(string(in.Bytes()))
			if nil != err {
				return edgex.NewMessageString("EX=ERR:" + err.Error())
			}
			// Write
			if err := tryWrite(conn, cmd, writeTimeout); nil != err {
				return edgex.NewMessageString("EX=ERR:" + err.Error())
			}
			// Read
			var n = int(0)
			for i := 0; i < 3; i++ {
				if n, err = tryRead(conn, buffer, readTimeout); nil != err {
					ctx.Log().Error("读取数据出错:", err)
					<-time.After(time.Second)
				} else {
					break
				}
			}
			// parse
			if n > 0 {
				if retCmd, err := dongk.ParseCommand(buffer); nil != err {
					return edgex.NewMessageString("EX=ERR:" + err.Error())
				} else if retCmd.Data[0] == 0x01 {
					return edgex.NewMessageString(fmt.Sprintf("EX=OK"))
				} else {
					return edgex.NewMessageString("EX=ERR:NOT_OK")
				}
			} else {
				return edgex.NewMessageString("EX=ERR:NO_REPLY")
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

func registryAt(registry *at.AtRegister, serialNumber uint32) {
	// AT+OPEN=SWITCH_ID
	registry.Add("OPEN", func(args ...string) (data []byte, err error) {
		switchId, err := strconv.ParseInt(args[0], 10, 64)
		if nil != err {
			return nil, errors.New("INVALID_SWITCH_ID:" + args[0])
		}
		return dongk.NewCommand(dongk.DkFunIdRemoteOpen,
				serialNumber,
				0,
				[32]byte{byte(switchId)}).Bytes(),
			nil
	})
	// AT+DELAY=SWITCH_ID,DELAY_SEC
	registry.Add("DELAY", func(args ...string) (data []byte, err error) {
		switchId, err := strconv.ParseInt(args[0], 10, 64)
		if nil != err {
			return nil, errors.New("INVALID_SWITCH_ID:" + args[0])
		}
		sec, err := strconv.ParseInt(args[1], 10, 64)
		if nil != err {
			return nil, errors.New("INVALID_DELAY_SEC:" + args[1])
		}
		return dongk.NewCommand(dongk.DkFunIdSwitchDelay,
				serialNumber,
				0,
				[32]byte{byte(switchId), byte(sec)}).Bytes(),
			nil
	})
}
