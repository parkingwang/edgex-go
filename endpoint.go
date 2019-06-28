package edgex

import (
	"context"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"net"
	"sync"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// Endpoint是接收、处理，并返回结果的可控制终端节点。
type Endpoint interface {
	Lifecycle
	NodeName

	// 处理RPC消息，返回处理结果
	Serve(func(in Message) (out Message))
}

type EndpointOptions struct {
	RpcAddr         string         // RPC 地址
	NodeName        string         // 节点名字
	SerialExecuting bool           // 是否串行地执行控制指令
	InspectFunc     func() Inspect // // Inspect消息生成函数
}

//// Endpoint实现

type NodeEndpoint struct {
	Endpoint
	nodeName        string
	globals         *Globals
	serialExecuting bool
	// Inspect
	inspectFunc func() Inspect
	// gRPC
	endpointAddr  string
	messageWorker func(in Message) (out Message)
	rpcServer     *grpc.Server
	// MQTT
	mqttClient mqtt.Client
	// Shutdown
	shutdownContext context.Context
	shutdownCancel  context.CancelFunc
}

func (e *NodeEndpoint) NodeName() string {
	return e.nodeName
}

func (e *NodeEndpoint) Startup() {
	e.shutdownContext, e.shutdownCancel = context.WithCancel(context.Background())

	e.rpcServer = grpc.NewServer()
	RegisterExecuteServer(e.rpcServer, &grpcExecutor{
		serialExecuting: e.serialExecuting,
		executeLock:     new(sync.Mutex),
		handler:         e.messageWorker,
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
	opts.SetClientID(fmt.Sprintf("EX-Endpoint:%s", e.nodeName))
	opts.SetWill(topicOfOffline("Endpoint", e.nodeName), "offline", 1, true)
	mqttSetOptions(opts, e.globals)
	e.mqttClient = mqtt.NewClient(opts)
	log.Info("Mqtt客户端连接Broker: ", e.globals.MqttBroker)

	// 连续重试
	mqttAwaitConnection(e.mqttClient, e.globals.MqttMaxRetry)

	if !e.mqttClient.IsConnected() {
		log.Panic("Mqtt客户端连接无法连接Broker")
	} else {
		// 异步发送Inspect消息
		mqttSendInspectMessage(e.mqttClient, e.nodeName, e.inspectFunc)
		go mqttAsyncTickInspect(e.shutdownContext, func() {
			mqttSendInspectMessage(e.mqttClient, e.nodeName, e.inspectFunc)
		})
	}
}

func (e *NodeEndpoint) Shutdown() {
	log.Info("停止GRPC服务")
	e.shutdownCancel()
	e.rpcServer.Stop()
	e.mqttClient.Disconnect(e.globals.MqttQuitMillSec)
}

func (e *NodeEndpoint) Serve(w func(in Message) (out Message)) {
	e.messageWorker = w
}

////

type grpcExecutor struct {
	ExecuteServer
	serialExecuting bool
	executeLock     *sync.Mutex
	handler         func(in Message) (out Message)
}

func (ex *grpcExecutor) Execute(c context.Context, i *Data) (o *Data, e error) {
	if ex.serialExecuting {
		ex.executeLock.Lock()
		defer ex.executeLock.Unlock()
		return ex.execute(c, i)
	} else {
		return ex.execute(c, i)
	}
}

func (ex *grpcExecutor) execute(c context.Context, i *Data) (o *Data, e error) {
	done := make(chan *Data, 1)
	in := ParseMessage(i.GetFrames())
	go func() {
		frames := ex.handler(in).getFrames()
		done <- &Data{Frames: frames}
	}()
	select {
	case v := <-done:
		return v, nil

	case <-c.Done():
		return nil, errors.New("GRPC_EXECUTE_TIMEOUT")
	}
}
