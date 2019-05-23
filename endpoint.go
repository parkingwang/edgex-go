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
	// 处理消息并返回
	Serve(func(in Message) (out Message))

	// 发送Alive消息
	NotifyAlive(state Message) error
}

type EndpointOptions struct {
	RpcAddr string
	Name    string
}

//// Endpoint实现

type implEndpoint struct {
	Endpoint
	name   string
	scoped *GlobalScoped
	// gRPC
	endpointAddr  string
	messageWorker func(in Message) (out Message)
	server        *grpc.Server
	// MQTT
	mqttClient mqtt.Client
}

func (e *implEndpoint) Startup() {
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
	// MQTT Broker
	opts := mqtt.NewClientOptions()
	opts.SetClientID(fmt.Sprintf("Endpoint-%s", e.name))
	opts.SetWill(topicOfOffline("Endpoint", e.name), "offline", 1, true)
	setMqttDefaults(opts, e.scoped)

	e.mqttClient = mqtt.NewClient(opts)
	log.Info("Mqtt客户端连接Broker: ", e.scoped.MqttBroker)

	// 连续重试
	for retry := 0; retry < e.scoped.MqttMaxRetry; retry++ {
		if token := e.mqttClient.Connect(); token.Wait() && token.Error() != nil {
			log.Error("Mqtt客户端连接出错：", token.Error())
			<-time.After(time.Second)
		} else {
			log.Info("Mqtt客户端连接成功: ")
			break
		}
	}
	if !e.mqttClient.IsConnected() {
		log.Panic("Mqtt客户端连接无法连接Broker")
	}
}

func (e *implEndpoint) Shutdown() {
	log.Info("停止GRPC服务")
	e.server.Stop()
	e.mqttClient.Disconnect(1000)
}

func (e *implEndpoint) Serve(w func(in Message) (out Message)) {
	e.messageWorker = w
}

func (e *implEndpoint) NotifyAlive(m Message) error {
	token := e.mqttClient.Publish(
		topicOfAlive("Endpoint", e.name),
		0,
		true,
		m.getFrames())
	if token.Wait() && nil != token.Error() {
		return errors.WithMessage(token.Error(), "发送Alive消息出错")
	} else {
		return nil
	}
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
