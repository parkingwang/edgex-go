package edgex

import (
	"bytes"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

const (
	FrameHeaderSize  = 2
	FrameVarBits     = 0xED
	FrameNameMaxSize = 127
)

// Header 头部
type Header struct {
	VarBits byte
	NameLen byte
}

// Message 消息
type Message interface {
	// getFrames 返回消息全部字节
	getFrames() []byte

	// Header 返回消息的Header
	Header() Header

	// Body 返回消息体字节
	Body() []byte

	// Name 返回消息体创建源组件名字
	Name() []byte

	// Size 返回Body消息体的大小
	Size() int
}

////

type implMessage struct {
	Message
	head []byte
	name []byte
	body []byte
}

func (m *implMessage) Name() []byte {
	return m.name
}

func (m *implMessage) Header() Header {
	return Header{
		m.head[0],
		m.head[1],
	}
}

func (m *implMessage) Body() []byte {
	return m.body
}

func (m *implMessage) getFrames() []byte {
	frames := new(bytes.Buffer)
	frames.Write(m.head)
	frames.Write(m.name)
	frames.Write(m.body)
	return frames.Bytes()
}

func (m *implMessage) Size() int {
	return len(m.body)
}

func NewMessageString(name, body string) Message {
	return NewMessage([]byte(name), []byte(body))
}

func NewMessage(name []byte, body []byte) Message {
	nameLen := len(name)
	if nameLen > FrameNameMaxSize {
		log.Panic("Name length too large, was: ", nameLen)
	}
	return &implMessage{
		head: []byte{FrameVarBits, byte(nameLen)},
		name: name,
		body: body,
	}
}

func ParseMessage(data []byte) Message {
	d := data[:FrameHeaderSize]
	head := Header{
		VarBits: d[0],
		NameLen: d[1],
	}
	return &implMessage{
		head: d,
		name: data[FrameHeaderSize : FrameHeaderSize+head.NameLen],
		body: data[FrameHeaderSize+head.NameLen:],
	}
}
