package dongk

import (
	"encoding/binary"
	"errors"
	"github.com/yoojia/edgex"
	"strconv"
)

//
// Author: 陈哈哈 chenyongjia@parkingwang.com, yoojiachen@gmail.com
//

const (
	// 东控Magic位
	DkMagic = 0x19
	// 主板状态
	DkFunIdBoardState = 0x20
	// 远程开门
	DkFunIdRemoteOpen = 0x40
	// 设置开关延时
	DkFunIdSwitchDelay = 0x80
)

// 开关控制方式
const (
	DkSwitchDelayAlwaysOpen   = 0x01 // 常开
	DkSwitchDelayAlwaysClose  = 0x02 // 常闭
	DkSwitchDelayAlwaysOnline = 0x03 // 在线控制，默认方式
)

// 东控门禁主板指令
type Command struct {
	magic    byte     // 1
	funcId   byte     // 1
	reversed uint16   // 2
	sn       uint32   // 4
	data     [32]byte // 32
	seqId    uint32   // 4
	extra    [20]byte // 20
}

func (dk *Command) FuncId() byte {
	return dk.funcId
}

func (dk *Command) SerialNum() uint32 {
	return dk.sn
}

func (dk *Command) Data() [32]byte {
	return dk.data
}

func (dk *Command) SeqId() uint32 {
	return dk.seqId
}

func (dk *Command) Bytes() []byte {
	// 东控数据包使用小字节序
	br := edgex.NewByteWriter(binary.LittleEndian)
	br.PutByte(dk.magic)
	br.PutByte(dk.funcId)
	br.PutUint16(dk.reversed)
	br.PutUint32(dk.sn)
	br.PutBytes(dk.data[:])
	br.PutUint32(dk.seqId)
	br.PutBytes(dk.extra[:])
	return br.Bytes()
}

// 创建DK指令
func NewCommand0(magic, funcId byte, nop uint16, serial uint32, seqId uint32, data [32]byte, extra [20]byte) *Command {
	return &Command{
		magic:    magic,
		funcId:   funcId,
		reversed: nop,
		sn:       serial,
		data:     data,
		seqId:    seqId,
		extra:    extra,
	}
}

// 创建DK指令
func NewCommand(funcId byte, serialId uint32, seqId uint32, data [32]byte) *Command {
	return NewCommand0(
		DkMagic,
		funcId,
		0x00,
		serialId,
		seqId,
		data,
		[20]byte{})
}

// 解析DK数据指令。如果数据指令长度不是64位，返回错误
func ParseCommand(frame []byte) (*Command, error) {
	size := len(frame)
	if 64 != size {
		return nil, errors.New("东控指令帧长度错误:" + strconv.FormatInt(int64(size), 10))
	}
	br := edgex.WrapByteReader(frame, binary.LittleEndian)
	magic := br.GetByte()
	funId := br.GetByte()
	reserved := br.GetUint16()
	serial := br.GetUint32()
	data := [32]byte{}
	copy(data[:], br.GetBytesSize(32))
	seqId := br.GetUint32()
	extra := [20]byte{}
	copy(extra[:], br.GetBytesSize(20))
	return NewCommand0(
		magic,
		funId,
		reserved,
		serial,
		seqId,
		data,
		extra,
	), nil
}
