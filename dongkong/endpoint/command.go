package main

import (
	"encoding/binary"
	"errors"
	"github.com/nextabc-lab/edgex"
	"github.com/nextabc-lab/edgex/dongkong"
	"github.com/yoojia/go-at"
	"strconv"
)

//
// Author: 陈哈哈 chenyongjia@parkingwang.com, yoojiachen@gmail.com
//

func atCommands(registry *at.AtRegister, serialNumber uint32) {
	// AT+OPEN=SWITCH_ID
	registry.AddX("OPEN", 1, func(args ...string) (data []byte, err error) {
		switchId, err := parseInt(args[0])
		if nil != err {
			return nil, errors.New("INVALID_SWITCH_ID:" + args[0])
		}
		return dongk.NewCommand(dongk.FunIdRemoteOpen,
				serialNumber,
				0,
				[32]byte{byte(switchId)}).Bytes(),
			nil
	})
	// AT+DELAY=SWITCH_ID,DELAY_SEC
	registry.AddX("DELAY", 2, func(args ...string) (data []byte, err error) {
		switchId, err := parseInt(args[0])
		if nil != err {
			return nil, errors.New("INVALID_SWITCH_ID:" + args[0])
		}
		seconds, err := parseInt(args[1])
		if nil != err {
			return nil, errors.New("INVALID_DELAY_SEC:" + args[1])
		}
		return dongk.NewCommand(dongk.FunIdSwitchDelay,
				serialNumber,
				0,
				[32]byte{
					byte(switchId),                // 门号
					dongk.SwitchDelayAlwaysOnline, // 控制方式
					byte(seconds),                 //开门延时：秒
				}).Bytes(),
			nil
	})
	// AT+ADD=CARD(uint32),START_DATE(YYYYMMdd),END_DATE(YYYYMMdd),DOOR1,DOOR2,DOOR3,DOOR4

	// AT+DELETE=CARD(uint32)
	registry.AddX("DELETE", 1, func(args ...string) (out []byte, err error) {
		card, err := parseInt(args[0])
		if nil != err {
			return nil, errors.New("INVALID_CARD:" + args[0])
		}
		data := [32]byte{}
		w := edgex.WrapByteWriter(data[:], binary.LittleEndian)
		w.PutUint32(uint32(card))
		return dongk.NewCommand(dongk.FunIdSwitchDelay,
				serialNumber,
				0,
				data).Bytes(),
			nil
	})

	// AT+CLEAR
	registry.AddX("CLEAR", 0, func(args ...string) (data []byte, err error) {
		return dongk.NewCommand(dongk.FunIdCardClear,
				serialNumber,
				0,
				[32]byte{0x55, 0xAA, 0xAA, 0x55}).Bytes(),
			nil
	})
}

func parseInt(val string) (int64, error) {
	return strconv.ParseInt(val, 10, 64)
}
