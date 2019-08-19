package edgex

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
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

const (
	eventIdByteSize = 8
)

var (
	ErrInvalidMessageLength = errors.New("INVALID_MESSAGE:LENGTH")
	ErrInvalidMessageHeader = errors.New("INVALID_MESSAGE:HEADER")
)

// Header 头部
type Header struct {
	Magic      byte  // Magic字段，固定为 0xED
	Version    byte  // 协议版本
	ControlVar byte  // 控制变量
	EventId    int64 // 消息事件ID，具有唯一性
}

// Message 消息接口。
type Message interface {
	// Header 返回消息的Header
	Header() Header

	// NodeId 返回节点ID
	NodeId() string

	// GroupId 返回组ID
	GroupId() string

	// MajorId 返回主ID
	MajorId() string

	// MinorId 返回次ID
	MinorId() string

	// UnionId 返回 NodeId:GroupId:MajorId:MinorId 的组合ID
	UnionId() string

	// EventId 返回消息Id。
	// 消息ID使用SnowflakeID生成器具有唯一性。
	EventId() int64

	// Body 返回消息体字节
	Body() []byte

	// Bytes 返回消息对象全部字节
	Bytes() []byte
}

////

type message struct {
	Message
	header   *Header
	unionId  string
	_unionId []string
	body     []byte
}

func (m *message) UnionId() string {
	return m.unionId
}

func (m *message) NodeId() string {
	return m._unionId[0]
}

func (m *message) GroupId() string {
	return m._unionId[1]
}

func (m *message) MajorId() string {
	return m._unionId[2]
}

func (m *message) MinorId() string {
	return m._unionId[3]
}

func (m *message) Header() Header {
	return *m.header
}

func (m *message) EventId() int64 {
	return m.header.EventId
}

func (m *message) Body() []byte {
	return m.body
}

func (m *message) Bytes() []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(m.header.Magic)
	buf.WriteByte(m.header.Version)
	buf.WriteByte(m.header.ControlVar)
	buf.Write(encodeInt64(m.header.EventId))
	buf.WriteString(m.unionId)
	buf.WriteByte(FrameEmpty)
	buf.Write(m.body)
	return buf.Bytes()
}

func splitUnionId(unionId string) []string {
	_unionId := strings.Split(unionId, ":")
	remains := 4 - len(_unionId)
	for i := 0; i < remains; i++ {
		_unionId = append(_unionId, "")
	}
	return _unionId[:4]
}

// 创建消息对象
func NewMessageByUnionId(unionId string, bodyBytes []byte, eventId int64) Message {
	return &message{
		header: &Header{
			Magic:      FrameMagic,
			Version:    FrameVersion,
			ControlVar: FrameVarData,
			EventId:    eventId,
		},
		unionId:  unionId,
		_unionId: splitUnionId(unionId),
		body:     bodyBytes,
	}
}

// 创建消息对象
func NewMessage(nodeId, groupId, majorId, minorId string, bodyBytes []byte, eventId int64) Message {
	return NewMessageByUnionId(
		MakeUnionId(nodeId, groupId, majorId, minorId),
		bodyBytes,
		eventId)
}

// 解析消息对象
func ParseMessage(data []byte) Message {
	reader := bufio.NewReader(bytes.NewReader(data))
	magic, _ := reader.ReadByte()
	version, _ := reader.ReadByte()
	vars, _ := reader.ReadByte()
	eventId := make([]byte, eventIdByteSize)
	if _, err := reader.Read(eventId); nil != err {
		panic(err)
	}
	uid, _ := reader.ReadBytes(FrameEmpty)
	body := make([]byte, len(data)-(3 /*Magic+Ver+Var*/ +eventIdByteSize)-len(uid))
	if _, err := reader.Read(body); nil != err {
		panic(err)
	}
	unionId := string(uid[:len(uid)-1])
	return &message{
		header: &Header{
			Magic:      magic,
			Version:    version,
			ControlVar: vars,
			EventId:    decodeInt64(eventId),
		},
		unionId:  unionId,
		_unionId: splitUnionId(unionId),
		body:     body,
	}
}

func MakeUnionId(nodeId, groupId, majorId, minorId string) string {
	checkRequiredId(nodeId, "nodeId")
	checkRequiredId(groupId, "groupId")
	checkRequiredId(majorId, "majorId")
	return nodeId + ":" + groupId + ":" + majorId + ":" + minorId
}

func encodeInt64(num int64) []byte {
	bs := make([]byte, eventIdByteSize)
	binary.BigEndian.PutUint64(bs, uint64(num))
	return bs
}

func decodeInt64(bs []byte) int64 {
	if nil == bs || 0 == len(bs) {
		return 0
	} else {
		return int64(binary.BigEndian.Uint64(bs))
	}
}
