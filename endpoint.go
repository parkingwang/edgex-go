package edgex

import (
	"context"
	"fmt"
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
	Lifecycle
	NodeName

	// 处理RPC消息，返回处理结果
	Serve(func(in Message) (out Message))

	// NextSequenceId 返回流水号
	NextSequenceId() uint32

	// 基于内部流水号创建消息对象
	NextMessage(virtualNodeId string, body []byte) Message
}

type EndpointOptions struct {
	RpcAddr         string          // RPC 地址
	NodeName        string          // 节点名字
	SerialExecuting bool            // 是否串行地执行控制指令
	InspectNodeFunc func() MainNode // // Inspect消息生成函数
}

//// Endpoint实现

type endpoint struct {
	Endpoint
	nodeName        string
	globals         *Globals
	sequenceId      uint32
	serialExecuting bool
	// MainNode
	inspectNodeFunc func() MainNode
	// gRPC
	endpointAddr string
	handler      func(in Message) (out Message)
	rpcServer    *grpc.Server
	// MQTT
	mqttClient mqtt.Client
	// Shutdown
	shutdownContext context.Context
	shutdownCancel  context.CancelFunc
}

func (e *endpoint) NodeName() string {
	return e.nodeName
}

func (e *endpoint) NextSequenceId() uint32 {
	e.sequenceId = (e.sequenceId + 1) % math.MaxUint32
	return e.sequenceId
}

func (e *endpoint) NextMessage(virtualNodeId string, body []byte) Message {
	return NewMessageWithId(e.nodeName, virtualNodeId, body, e.NextSequenceId())
}

func (e *endpoint) Startup() {
	e.shutdownContext, e.shutdownCancel = context.WithCancel(context.Background())

	e.rpcServer = grpc.NewServer()
	RegisterExecuteServer(e.rpcServer, &grpcExecutor{
		nodeName:        e.nodeName,
		serialExecution: e.serialExecuting,
		serialLock:      new(sync.Mutex),
		handler:         e.handler,
		nextSequenceId:  e.NextSequenceId,
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
	// MQTT Broker
	opts := mqtt.NewClientOptions()
	clientId := fmt.Sprintf("EX-Endpoint:%s", e.nodeName)
	opts.SetClientID(clientId)
	opts.SetWill(topicOfOffline("Endpoint", e.nodeName), "offline", 1, true)
	mqttSetOptions(opts, e.globals)
	e.mqttClient = mqtt.NewClient(opts)
	log.Info("Mqtt客户端连接Broker: ", e.globals.MqttBroker)

	// 连续重试
	mqttAwaitConnection(e.mqttClient, e.globals.MqttMaxRetry)

	if !e.mqttClient.IsConnected() {
		log.Panic("Mqtt客户端连接无法连接Broker")
	} else {
		log.Debug("Mqtt客户端连接成功：" + clientId)
		// 异步发送Inspect消息
		mqttSendInspectMessage(e.mqttClient, e.nodeName, e.inspectNodeFunc)
		// 定时发送
		go mqttAsyncTickInspect(e.shutdownContext, func() {
			mqttSendInspectMessage(e.mqttClient, e.nodeName, e.inspectNodeFunc)
		})
	}
}

func (e *endpoint) Shutdown() {
	log.Info("停止GRPC服务")
	e.shutdownCancel()
	e.rpcServer.Stop()
	e.mqttClient.Disconnect(e.globals.MqttQuitMillSec)
}

func (e *endpoint) Serve(h func(in Message) (out Message)) {
	e.handler = h
}

////

type grpcExecutor struct {
	ExecuteServer
	nodeName        string
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
		pong := newControlMessageWithId(ex.nodeName, ex.nodeName, FrameVarPong, ex.nextSequenceId())
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
