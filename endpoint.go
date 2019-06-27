package edgex

import (
	"context"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"net"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// Endpoint是接收、处理，并返回结果的可控制终端节点。
type Endpoint interface {
	Lifecycle
	NodeName

	// 处理RPC消息并返回处理结果
	Serve(func(in Message) (out Message))
}

type EndpointOptions struct {
	RpcAddr     string         // RPC 地址
	NodeName    string         // 节点名字
	InspectFunc func() Inspect // // Inspect消息生成函数
}

//// Endpoint实现

type NodeEndpoint struct {
	Endpoint
	nodeName string
	globals  *Globals
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
	RegisterExecuteServer(e.rpcServer, &executor{
		handler: e.messageWorker,
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

type executor struct {
	ExecuteServer
	handler func(in Message) (out Message)
}

func (ex *executor) Execute(c context.Context, i *Data) (o *Data, e error) {
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
		return nil, errors.New("execute timeout")
	}
}
