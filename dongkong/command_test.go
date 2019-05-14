package dongk

import (
	"encoding/hex"
	"fmt"
	"testing"
)

//
// Author: 陈哈哈 chenyongjia@parkingwang.com, yoojiachen@gmail.com
//

func TestDKCommand_Bytes(t *testing.T) {

	dk := NewCommand(0x40,
		223177933,
		23,
		[32]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF})

	rawBytes := dk.Bytes()
	fmt.Printf("%X\n", rawBytes)

	reDK, err := ParseCommand(rawBytes)
	if nil != err {
		panic(err)
	}

	reBytes := reDK.Bytes()
	fmt.Printf("%X", reBytes)

	if hex.EncodeToString(rawBytes) != hex.EncodeToString(reBytes) {
		t.Fatal("Not match")
	}
}

func TestParseCommand(t *testing.T) {
	dk, err := ParseCommand([]byte{
		0x17,
		0x40,
		0x00, 0x00,
		0x1D, 0x85, 0xB5, 0x0D,
		0x01,
	})
	if nil != err {
		t.Error(err)
	}
	if dk.Magic != 0x17 {
		t.Error("Magic not match: ", dk.Magic)
	}
	if dk.SerialNum != 229999901 {
		t.Error("SN not match:", dk.SerialNum)
	}
	if !dk.Success() {
		t.Error("Should success")
	}
}
