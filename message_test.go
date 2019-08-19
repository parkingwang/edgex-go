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

func TestSplitUnionId(t *testing.T) {
	check := func(unionId []string, idx int, excepted string) {
		if 4 != len(unionId) || excepted != unionId[idx] {
			t.Errorf("Not match, except: %s, was: %s", excepted, unionId[idx])
		}
	}

	u1 := splitUnionId("")
	check(u1, 0, "")
	check(u1, 2, "")
	check(u1, 1, "")
	check(u1, 3, "")

	u2 := splitUnionId("a")
	check(u2, 0, "a")
	check(u2, 1, "")

	u3 := splitUnionId("a:")
	check(u3, 0, "a")
	check(u3, 1, "")

	u4 := splitUnionId("a:b")
	check(u4, 0, "a")
	check(u4, 1, "b")
	check(u4, 2, "")

	u5 := splitUnionId("a:b:c:d")
	check(u5, 0, "a")
	check(u5, 1, "b")
	check(u5, 2, "c")
	check(u5, 3, "d")

	u6 := splitUnionId("a:b:c:d:e:f")
	check(u6, 0, "a")
	check(u6, 1, "b")
	check(u6, 2, "c")
	check(u6, 3, "d")

	u7 := splitUnionId("a::c:d")
	check(u7, 0, "a")
	check(u7, 1, "")
	check(u7, 2, "c")
	check(u7, 3, "d")
}

func TestNewMessage(t *testing.T) {
	body := []byte{FrameEmpty, 0xAA, FrameEmpty, 0xBB, 0xCC, FrameEmpty}
	newMessage := NewMessage("CHEN", "NODE", "A", "", body, 2019)

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
		if header.EventId != 2019 || msg.EventId() != 2019 {
			t.Error("EventId var not match, was: ", msg.EventId())
		}
		if MakeUnionId("CHEN", "NODE", "A", "") != msg.UnionId() {
			t.Error("UnionId not match, was: ", msg.UnionId())
		}
		if !bytes.Equal(body, msg.Body()) {
			t.Error("Body not match, was", hex.EncodeToString(msg.Body()))
		}
	}

	fmt.Println("Bytes: " + hex.EncodeToString(newMessage.Bytes()))
	check(newMessage)

	// Parse
	parsed := ParseMessage(newMessage.Bytes())

	check(parsed)
}
