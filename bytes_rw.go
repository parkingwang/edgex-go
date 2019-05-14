package edgex

import (
	"bytes"
	"encoding/binary"
	"io"
)

//
// Author: 陈哈哈 chenyongjia@parkingwang.com, yoojiachen@gmail.com
//

type wrapper struct {
	order  binary.ByteOrder
	buffer *bytes.Buffer
}

func (r *wrapper) Reset() {
	r.buffer.Reset()
}

////

// 字节Reader，提供从数据帧顺序读取类型数据的函数。
type ByteReader struct {
	*wrapper
}

func WrapByteReader(frame []byte, order binary.ByteOrder) *ByteReader {
	return &ByteReader{
		wrapper: &wrapper{
			order:  order,
			buffer: bytes.NewBuffer(frame),
		},
	}
}

func (r *ByteReader) GetByte() byte {
	b, _ := r.buffer.ReadByte()
	return b
}

func (r *ByteReader) LoadBytes(out []byte) (n int, err error) {
	return r.buffer.Read(out)
}

func (r *ByteReader) GetBytesSize(size int) []byte {
	out := make([]byte, size)
	if _, err := r.LoadBytes(out); nil != err && err != io.EOF {
		panic(err)
	}
	return out
}

func (r *ByteReader) GetUint16() uint16 {
	return r.order.Uint16(r.GetBytesSize(2))
}

func (r *ByteReader) GetUint32() uint32 {
	return r.order.Uint32(r.GetBytesSize(4))
}

func (r *ByteReader) GetUint64() uint64 {
	return r.order.Uint64(r.GetBytesSize(8))
}

////

// 字节Reader，提供向数据缓存顺序写入类型数据的函数。
type ByteWriter struct {
	*wrapper
}

func NewByteWriter(order binary.ByteOrder) *ByteWriter {
	return WrapByteBufferWriter(bytes.NewBuffer(make([]byte, 0)), order)
}

func WrapByteWriter(buffer []byte, order binary.ByteOrder) *ByteWriter {
	return WrapByteBufferWriter(bytes.NewBuffer(buffer), order)
}

func WrapByteBufferWriter(buffer *bytes.Buffer, order binary.ByteOrder) *ByteWriter {
	return &ByteWriter{
		wrapper: &wrapper{
			order:  order,
			buffer: buffer,
		},
	}
}

func (w *ByteWriter) PutByte(b byte) {
	w.buffer.WriteByte(b)
}

func (w *ByteWriter) PutBytes(bs []byte) {
	w.buffer.Write(bs)
}

func (w *ByteWriter) PutUint16(value uint16) {
	b := make([]byte, 2)
	w.order.PutUint16(b, value)
	w.buffer.Write(b)
}

func (w *ByteWriter) PutUint32(value uint32) {
	b := make([]byte, 4)
	w.order.PutUint32(b, value)
	w.buffer.Write(b)
}

func (w *ByteWriter) PutUint64(value uint64) {
	b := make([]byte, 8)
	w.order.PutUint64(b, value)
	w.buffer.Write(b)
}

func (w *ByteWriter) Bytes() []byte {
	return w.buffer.Bytes()
}
