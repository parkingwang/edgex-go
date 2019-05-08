package edgex

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// Packet消息
type Packet []byte

func PacketOfString(txt string) Packet {
	return Packet([]byte(txt))
}

func PacketOfBytes(b []byte) Packet {
	return Packet(b)
}

func (p Packet) Bytes() []byte {
	return []byte(p)
}
