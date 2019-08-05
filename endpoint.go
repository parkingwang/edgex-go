package edgex

import (
	"context"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"math"
	"net"
	"sync"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

var (
	ErrGRPCExecTimeout = errors.New("GRPC_EXEC_TIMEOUT")
	ErrUnknownMessage  = errors.New("UNKNOWN_MESSAGE")
)

// Endpoint是接收、处理，并返回结果的可控制终端节点。
type Endpoint interface {
	NeedLifecycle
	NeedNodeId
	NeedMessages

	// 处理RPC消息，返回处理结果
	Serve(func(in Message) (out Message))

	// 发布Inspect消息
	PublishInspectMessage(node MainNode)
}

type EndpointOptions struct {
	RpcAddr         string          // RPC 地址
	SerialExecuting bool            // 是否串行地执行控制指令
	AutoInspectFunc func() MainNode // // Inspect消息生成函数
}

//// Endpoint实现

type endpoint struct {
	Endpoint
	nodeId          string
	globals         *Globals
	sequenceId      uint32
	serialExecuting bool
	// MainNode
	autoInspectFunc func() MainNode
	// gRPC
	endpointAddr string
	handler      func(in Message) (out Message)
	rpcServer    *grpc.Server
	// MQTT
	refMqttClient mqtt.Client
	// Shutdown
	shutdownContext context.Context
	shutdownCancel  context.CancelFunc
}

func (e *endpoint) NodeId() string {
	return e.nodeId
}

func (e *endpoint) NextMessageSequenceId() uint32 {
	e.sequenceId = (e.sequenceId + 1) % math.MaxUint32
	return e.sequenceId
}

func (e *endpoint) NextMessageByVirtualId(virtualId string, body []byte) Message {
	return NewMessageByVirtualId(e.nodeId, virtualId, body, e.NextMessageSequenceId())
}

func (e *endpoint) NextMessageBySourceUuid(sourceUuid string, body []byte) Message {
	return NewMessageBySourceUuid(sourceUuid, body, e.NextMessageSequenceId())
}

func (e *endpoint) Startup() {
	e.shutdownContext, e.shutdownCancel = context.WithCancel(context.Background())

	e.rpcServer = grpc.NewServer()
	RegisterExecuteServer(e.rpcServer, &grpcExecutor{
		nodeId:          e.nodeId,
		serialExecution: e.serialExecuting,
		serialLock:      new(sync.Mutex),
		handler:         e.handler,
		nextSequenceId:  e.NextMessageSequenceId,
	})
	// gRpc Serve
	go func() {
		listen, err := net.Listen("tcp", e.endpointAddr)
		if nil != err {
			log.Panic("Endpoint listen failed: ", err)
		}
		log.Info("开启GRPC服务: ", e.endpointAddr)
		if err := e.rpcServer.Serve(listen); nil != err {
			log.Error("GRPC server stop: ", err)
		}
	}()
	// 定时发送Inspect消息
	if nil != e.autoInspectFunc {
		go scheduleSendInspect(e.shutdownContext, func() {
			e.PublishInspectMessage(e.autoInspectFunc())
		})
	}
}

func (e *endpoint) PublishInspectMessage(node MainNode) {
	mqttSendInspectMessage(e.refMqttClient, e.nodeId, node)
}

func (e *endpoint) Shutdown() {
	log.Info("停止GRPC服务")
	e.shutdownCancel()
	e.rpcServer.Stop()
}

func (e *endpoint) Serve(h func(in Message) (out Message)) {
	e.handler = h
}

////

type grpcExecutor struct {
	ExecuteServer
	nodeId          string
	nextSequenceId  func() uint32
	serialExecution bool        // 是否串行执行
	serialLock      *sync.Mutex // 串行锁
	handler         func(in Message) (out Message)
}

func (ex *grpcExecutor) Execute(c context.Context, i *Data) (o *Data, e error) {
	if ex.serialExecution {
		ex.serialLock.Lock()
		defer ex.serialLock.Unlock()
		return ex.execute(c, i)
	} else {
		return ex.execute(c, i)
	}
}

func (ex *grpcExecutor) execute(c context.Context, i *Data) (o *Data, e error) {
	frames := i.GetFrames()
	if ok, err := CheckMessage(frames); !ok {
		return nil, err
	}
	in := ParseMessage(frames)
	switch in.Header().ControlVar {
	// Ping Pong
	case FrameVarPing:
		pong := newControlMessage(ex.nodeId, ex.nodeId, FrameVarPong, ex.nextSequenceId())
		return &Data{Frames: pong.Bytes()}, nil

	// Data
	case FrameVarData:
		done := make(chan *Data, 1)
		go func() {
			frames := ex.handler(in).Bytes()
			done <- &Data{Frames: frames}
		}()
		select {
		case v := <-done:
			return v, nil

		case <-c.Done():
			return nil, ErrGRPCExecTimeout
		}

	default:
		return nil, ErrUnknownMessage
	}
}
