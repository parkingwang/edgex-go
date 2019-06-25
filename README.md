# Edge X - Golang

`EdgeX-Go` 是基于Golang的，以MQTT消息服务和gRPC为核心的边缘计算框架项目。
可以将众多用户设备，以独立进程的方式接入到边缘计算服务。

EdgeX-Go的设计目标是为硬件设备提供通讯层框架，支持硬件设备接入、消息路由等能力。

## 主要功能

1. 提供基于静态配置文件的硬件配置及管理能力；
2. 提供基于MQTT的事件上行通讯、事件监听下行和事件处理的能力；
3. 提供基于gRPC的RPC主动指令点对点控制能力；
4. 提供跨平台编译和运行能力；

## 待实现能力

1. 基于MQTT来实现对实时性要求不高的RPC通讯能力。

这个功能已有Java版本的实现方案：[NextMQTT](https://gitee.com/bitschen/NextMQTT)。还需要GoLang版本的实现。


## 主要概念

**Trigger - 触发器**

Trigger的特点是主动上报设备状态数据。

**Endpoint - 执行终端**

Endpoint的特点是，被动接受Driver发起的控制指令，处理后，返回指令操作结果。

**Driver - 驱动**

Driver的特点是，监听Trigger触发的事件消息，经内部业务逻辑处理后，向Endpoint发起控制指令来完成设备控制动作。




