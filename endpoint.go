package edgex

import (
	ctx "context"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"net"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// Endpoint是接收、处理，并返回结果的可控制终端节点。
type Endpoint interface {
	Lifecycle
	// 处理RPC消息并返回处理结果
	Serve(func(in Message) (out Message))
	// 发送Alive消息
	SendAliveMessage(alive Message) error
}

type EndpointOptions struct {
	RpcAddr     string         // RPC 地址
	Name        string         // 名字
	InspectFunc func() Inspect // 返回Inspect数据的函数
}

//// Endpoint实现

type implEndpoint struct {
	Endpoint
	name          string
	scoped        *GlobalScoped
	inspectFunc   func() Inspect
	inspectTicker *time.Ticker
	// gRPC
	endpointAddr  string
	messageWorker func(in Message) (out Message)
	rpcServer     *grpc.Server
	// MQTT
	mqttClient mqtt.Client
}

func (e *implEndpoint) Startup() {
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
	opts.SetClientID(fmt.Sprintf("Endpoint-%s", e.name))
	opts.SetWill(topicOfOffline("Endpoint", e.name), "offline", 1, true)
	mqttSetOptions(opts, e.scoped)
	e.mqttClient = mqtt.NewClient(opts)
	log.Info("Mqtt客户端连接Broker: ", e.scoped.MqttBroker)
	// 连续重试
	mqttRetryConnect(e.mqttClient, e.scoped.MqttMaxRetry)
	if !e.mqttClient.IsConnected() {
		log.Panic("Mqtt客户端连接无法连接Broker")
	} else {
		mqttSendInspectMessage(e.mqttClient, e.name, e.inspectFunc)
		go func() {
			e.inspectTicker = time.NewTicker(time.Minute)
			for range e.inspectTicker.C {
				mqttSendInspectMessage(e.mqttClient, e.name, e.inspectFunc)
			}
		}()
	}
}

func (e *implEndpoint) Shutdown() {
	log.Info("停止GRPC服务")
	e.inspectTicker.Stop()
	e.rpcServer.Stop()
	e.mqttClient.Disconnect(1000)
}

func (e *implEndpoint) Serve(w func(in Message) (out Message)) {
	e.messageWorker = w
}

func (e *implEndpoint) SendAliveMessage(alive Message) error {
	return mqttSendAliveMessage(e.mqttClient, "Endpoint", e.name, alive)
}

////

type executor struct {
	ExecuteServer
	handler func(in Message) (out Message)
}

func (ex *executor) Execute(c ctx.Context, i *Data) (o *Data, e error) {
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
