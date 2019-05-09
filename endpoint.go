package edgex

import (
	ctx "context"
	"errors"
	"google.golang.org/grpc"
	"net"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// Endpoint是接收、处理，并返回结果的可控制终端节点。
type Endpoint interface {
	Lifecycle
	// 处理消息并返回
	Serve(func(in Packet) (out Packet))
}

type EndpointOptions struct {
	Addr string
	Name string
}

//// Endpoint实现

type endpoint struct {
	Endpoint
	scoped        *GlobalScoped
	endpointAddr  string
	messageWorker func(in Packet) (out Packet)
	server        *grpc.Server
}

func (e *endpoint) Startup() {
	e.server = grpc.NewServer()
	RegisterExecuteServer(e.server, &executor{
		handler: e.messageWorker,
	})
	go func() {
		listen, err := net.Listen("tcp", e.endpointAddr)
		if nil != err {
			log.Panic("Endpoint listen failed: ", err)
		}
		log.Info("开启GRPC服务: ", e.endpointAddr)
		if err := e.server.Serve(listen); nil != err {
			log.Error("GRPC server stop: ", err)
		}
	}()
}

func (e *endpoint) Shutdown() {
	log.Info("开启GRPC服务")
	e.server.Stop()
}

func (e *endpoint) Serve(w func(in Packet) (out Packet)) {
	e.messageWorker = w
}

////

type executor struct {
	ExecuteServer
	handler func(in Packet) (out Packet)
}

func (ex *executor) Execute(c ctx.Context, i *Data) (o *Data, e error) {
	done := make(chan *Data, 1)
	select {
	case done <- &Data{Frames: ex.handler(PacketOfBytes(i.Frames))}:
		return <-done, nil

	case <-c.Done():
		return nil, errors.New("execute timeout")
	}
}
