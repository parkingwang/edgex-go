package main

import (
	"github.com/tarm/serial"
	"github.com/yoojia/go-value"
	"strings"
	"time"
)

//
// Author: 陈哈哈 chenyongjia@parkingwang.com, yoojiachen@gmail.com
//

func getSerialConfig(config map[string]interface{}, timeout time.Duration) *serial.Config {
	parity := serial.ParityNone
	switch strings.ToUpper(value.Of(config["parity"]).String()) {
	case "N", "NONE":
		parity = serial.ParityNone
	case "O", "ODD":
		parity = serial.ParityOdd
	case "E", "EVEN":
		parity = serial.ParityEven
	case "M", "MARK":
		parity = serial.ParityMark
	case "S", "SPACE":
		parity = serial.ParitySpace

	default:
		parity = serial.ParityNone
	}
	return &serial.Config{
		Name:        value.Of(config["serialPort"]).String(),
		Baud:        int(value.Of(config["baudRate"]).MustInt64()),
		Size:        byte(value.Of(config["dataBit"]).MustInt64()),
		Parity:      parity,
		StopBits:    serial.StopBits(value.Of(config["stopBits"]).MustInt64()),
		ReadTimeout: timeout,
	}
}
