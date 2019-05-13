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

	data := [32]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}

	dk := NewCommand(0x40,
		223177933,
		23,
		data)

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
