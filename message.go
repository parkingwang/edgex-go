package edgex

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// Packet消息
type Message interface {
	Bytes() []byte
}

////

type implMessage struct {
	Message
	frames []byte
}

func (m *implMessage) Bytes() []byte {
	return m.frames
}

func NewMessageString(txt string) Message {
	return NewMessageBytes([]byte(txt))
}

func NewMessageBytes(b []byte) Message {
	return &implMessage{frames: b}
}
