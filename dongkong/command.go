package dongk

import (
	"encoding/binary"
	"errors"
	"github.com/nextabc-lab/edgex"
)

//
// Author: 陈哈哈 chenyongjia@parkingwang.com, yoojiachen@gmail.com
//

const (
	// 主板状态
	FunIdBoardState = 0x20
	// 远程开门
	FunIdRemoteOpen = 0x40
	// 设置开关延时
	FunIdSwitchDelay = 0x80
	// 添加卡
	FunIdCardAdd = 0x50
	// 删除卡
	FunIdCardDel = 0x52
	// 删除所有卡
	FunIdCardClear = 0x54
)

// 东控Magic位
const (
	Magic = 0x19
)

// 开关控制方式
const (
	SwitchDelayAlwaysOpen   = 0x01 // 常开
	SwitchDelayAlwaysClose  = 0x02 // 常闭
	SwitchDelayAlwaysOnline = 0x03 // 在线控制，默认方式
)

// 东控门禁主板指令
type Command struct {
	Magic     byte     // 1
	FuncId    byte     // 1
	reversed  uint16   // 2
	SerialNum uint32   // 4
	Data      [32]byte // 32
	SeqId     uint32   // 4
	Extra     [20]byte // 20
}

func (dk *Command) Bytes() []byte {
	// 东控数据包使用小字节序
	br := edgex.NewByteWriter(binary.LittleEndian)
	br.PutByte(dk.Magic)
	br.PutByte(dk.FuncId)
	br.PutUint16(dk.reversed)
	br.PutUint32(dk.SerialNum)
	br.PutBytes(dk.Data[:])
	br.PutUint32(dk.SeqId)
	br.PutBytes(dk.Extra[:])
	return br.Bytes()
}

// Success 返回接收报文的成功标记位状态
func (dk *Command) Success() bool {
	return 0x01 == dk.Data[0]
}

// 创建DK指令
func NewCommand0(magic, funcId byte, nop uint16, serial uint32, seqId uint32, data [32]byte, extra [20]byte) *Command {
	return &Command{
		Magic:     magic,
		FuncId:    funcId,
		reversed:  nop,
		SerialNum: serial,
		Data:      data,
		SeqId:     seqId,
		Extra:     extra,
	}
}

// 创建DK指令
func NewCommand(funcId byte, serialId uint32, seqId uint32, data [32]byte) *Command {
	return NewCommand0(
		Magic,
		funcId,
		0x00,
		serialId,
		seqId,
		data,
		[20]byte{})
}

// 解析DK数据指令。
func ParseCommand(frame []byte) (*Command, error) {
	if len(frame) < 9 {
		return nil, errors.New("invalid bytes len")
	}
	br := edgex.WrapByteReader(frame, binary.LittleEndian)
	magic := br.GetByte()
	funId := br.GetByte()
	reserved := br.GetUint16()
	serialNum := br.GetUint32()
	data := [32]byte{}
	copy(data[:], br.GetBytesSize(32))
	seqId := br.GetUint32()
	extra := [20]byte{}
	copy(extra[:], br.GetBytesSize(20))
	return NewCommand0(
		magic,
		funId,
		reserved,
		serialNum,
		seqId,
		data,
		extra,
	), nil
}

func DirectName(b byte) string {
	if 1 == b {
		return "IN"
	} else {
		return "OUT"
	}
}
