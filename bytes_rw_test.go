package edgex

import (
	"encoding/binary"
	"fmt"
	"testing"
)

func TestWrapByteWriterAndReader(t *testing.T) {

	bw := NewByteWriter(binary.LittleEndian)
	bw.PutByte(0xFF)
	bw.PutByte(0xEE)
	bw.PutBytes([]byte{0xAA, 0xBB, 0xCC})
	bw.PutUint16(2018)
	bw.PutUint32(228441083)
	bw.PutUint64(13800138000)

	allBytes := bw.Bytes()
	fmt.Printf("%X\n", allBytes)

	br := WrapByteReader(allBytes, binary.LittleEndian)

	if 0xFF != br.GetByte() {
		t.Fatal("FF")
	}

	if 0xEE != br.GetByte() {
		t.Fatal("EE")
	}

	bs := br.GetBytesSize(3)
	if 0xAA != bs[0] {
		t.Fatal("AA")
	}
	if 0xBB != bs[1] {
		t.Fatal("BB")
	}
	if 0xCC != bs[2] {
		t.Fatal("CC")
	}

	if 2018 != br.GetUint16() {
		t.Fatal("2018")
	}
	if 228441083 != br.GetUint32() {
		t.Fatal("228441083")
	}
	if 13800138000 != br.GetUint64() {
		t.Fatal("13800138000")
	}
}
