package edgex

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

func TestNewMessage(t *testing.T) {
	newMessage := NewMessageWithId("CHEN", []byte{0xAA, 0xBB, 0xCC}, 2019)

	check := func(msg Message) {
		header := msg.Header()
		if header.Magic != FrameMagic {
			t.Error("Magic not match")
		}
		if header.Version != FrameVersion {
			t.Error("Version not match")
		}
		if header.ControlVar != FrameVarData {
			t.Error("Control var not match")
		}
		if header.SequenceId != 2019 || msg.SequenceId() != 2019 {
			t.Error("SequenceId var not match, was: ", msg.SequenceId())
		}
		if "CHEN" != msg.SourceName() {
			t.Error("SourceName not match, was: ", msg.SourceName())
		}
		if !bytes.Equal([]byte{0xAA, 0xBB, 0xCC}, msg.Body()) {
			t.Error("Body not match, was", hex.EncodeToString(msg.Body()))
		}
	}

	fmt.Println("Bytes: " + hex.EncodeToString(newMessage.Bytes()))
	check(newMessage)

	// Parse
	parsed := ParseMessage(newMessage.Bytes())

	check(parsed)
}
