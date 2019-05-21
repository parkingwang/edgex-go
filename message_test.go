package edgex

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

func TestNewMessage(t *testing.T) {
	msg := NewMessage([]byte("CHEN"), []byte{0xAA, 0xBB, 0xCC})
	head := msg.Header()
	name := msg.Name()
	body := msg.Body()
	if head.VarBits != FrameVarBits {
		t.Error("Var Bits not match")
	}
	if head.NameLen != 4 {
		t.Error("Name len not match: ", head.NameLen)
	}
	if "CHEN" != string(name) {
		t.Error("Name not match: ", string(name))
	}
	if !reflect.DeepEqual([]byte{0xAA, 0xBB, 0xCC}, body) {
		t.Error("Body not match", body)
	}
	fmt.Println(hex.EncodeToString(msg.getFrames()))
}

func TestParseMessage(t *testing.T) {
	msg := ParseMessage([]byte{0xed, 0x04, 0x43, 0x48, 0x45, 0x4e, 0xaa, 0xbb, 0xcc})
	head := msg.Header()
	name := msg.Name()
	body := msg.Body()
	if head.VarBits != FrameVarBits {
		t.Error("Var Bits not match")
	}
	if head.NameLen != 4 {
		t.Error("Name len not match: ", head.NameLen)
	}
	if "CHEN" != string(name) {
		t.Error("Name not match, was: ", string(name))
	}
	if !reflect.DeepEqual([]byte{0xAA, 0xBB, 0xCC}, body) {
		t.Error("Body not match", body)
	}
}
