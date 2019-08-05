package edgex

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

const (
	FrameMagic   byte = 0xED // Magic
	FrameVersion      = 0x01 // 版本
	FrameEmpty        = 0x00 // 分隔空帧
	FrameVarData      = 0xDA
)

var (
	ErrInvalidMessageLength = errors.New("INVALID_MESSAGE:LENGTH")
	ErrInvalidMessageHeader = errors.New("INVALID_MESSAGE:HEADER")
)

// Header 头部
type Header struct {
	Magic      byte   // Magic字段，固定为 0xED
	Version    byte   // 协议版本
	ControlVar byte   // 控制变量
	SequenceId uint32 // 消息流水ID
}

// Message 消息接口。
type Message interface {
	// Header 返回消息的Header
	Header() Header

	// VirtualNodeId 虚拟节点ID
	// 它由两部分组成：NodeId + VirtualId。在可以唯一标识一个虚拟设备。
	VirtualNodeId() string

	// SequenceId 返回消息Id
	SequenceId() uint32

	// Body 返回消息体字节
	Body() []byte

	// Bytes 返回消息对象全部字节
	Bytes() []byte
}

////

type message struct {
	Message
	header        *Header
	virtualNodeId string
	body          []byte
}

func (m *message) VirtualNodeId() string {
	return m.virtualNodeId
}

func (m *message) Header() Header {
	return *m.header
}

func (m *message) SequenceId() uint32 {
	return m.header.SequenceId
}

func (m *message) Body() []byte {
	return m.body
}

func (m *message) Bytes() []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(m.header.Magic)
	buf.WriteByte(m.header.Version)
	buf.WriteByte(m.header.ControlVar)
	buf.Write(encodeUint32(m.header.SequenceId))
	buf.WriteString(m.virtualNodeId)
	buf.WriteByte(FrameEmpty)
	buf.Write(m.body)
	return buf.Bytes()
}

// 创建消息对象
func NewMessageById(virtualNodeId string, bodyBytes []byte, seqId uint32) Message {
	return &message{
		header: &Header{
			Magic:      FrameMagic,
			Version:    FrameVersion,
			ControlVar: FrameVarData,
			SequenceId: seqId,
		},
		virtualNodeId: virtualNodeId,
		body:          bodyBytes,
	}
}

// 创建消息对象
func NewMessageWith(nodeId, virtualId string, bodyBytes []byte, seqId uint32) Message {
	return NewMessageById(MakeVirtualNodeId(nodeId, virtualId), bodyBytes, seqId)
}

// 解析消息对象
func ParseMessage(data []byte) Message {
	reader := bufio.NewReader(bytes.NewReader(data))
	magic, _ := reader.ReadByte()
	version, _ := reader.ReadByte()
	vars, _ := reader.ReadByte()
	seqId := make([]byte, 4)
	if _, err := reader.Read(seqId); nil != err {
		panic(err)
	}
	vnId, _ := reader.ReadBytes(FrameEmpty)
	body := make([]byte, len(data)-7-len(vnId))
	if _, err := reader.Read(body); nil != err {
		panic(err)
	}
	return &message{
		header: &Header{
			Magic:      magic,
			Version:    version,
			ControlVar: vars,
			SequenceId: decodeUint32(seqId),
		},
		virtualNodeId: string(vnId[:len(vnId)-1]),
		body:          body,
	}
}

// 检查是否符合消息协议格式
func CheckMessage(data []byte) (bool, error) {
	if 8 > len(data) {
		return false, ErrInvalidMessageLength
	} else if FrameMagic != data[0] {
		return false, ErrInvalidMessageHeader
	} else if FrameVersion != data[1] {
		return false, ErrInvalidMessageHeader
	} else {
		return true, nil
	}
}

func MakeVirtualNodeId(nodeId, virtualId string) string {
	return nodeId + ":" + virtualId
}

func encodeUint32(num uint32) []byte {
	bs := make([]byte, 4)
	binary.BigEndian.PutUint32(bs, num)
	return bs
}

func decodeUint32(bs []byte) uint32 {
	if nil == bs || 0 == len(bs) {
		return 0
	} else {
		return binary.BigEndian.Uint32(bs)
	}
}
