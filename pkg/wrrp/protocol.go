package wrrp

import (
	"encoding/binary"
	"errors"
)

const (
	HeaderSize = 40
	// WRRP 的幻数 (4字节)
	MagicNumber uint32 = 0x57525250 // ASCII: 'W' 'R' 'R' 'P'
)

// 指令定义
const (
	Register uint8 = 0x01 // 客户端注册/握手
	Forward  uint8 = 0x02 // 数据转发
	Ping     uint8 = 0x03 // 心跳检测
)

// Header WRRP 协议头 (共 40 字节)
type Header struct {
	Magic      uint32
	Version    uint16
	Cmd        uint8
	Reserved   uint8
	PayloadLen uint32
	SessionID  [28]byte
}

// Marshal 将 Header 序列化为字节流
func (h *Header) Marshal() []byte {
	buf := make([]byte, HeaderSize)
	binary.BigEndian.PutUint32(buf[0:4], h.Magic)
	binary.BigEndian.PutUint16(buf[4:6], h.Version)
	buf[6] = h.Cmd
	buf[7] = h.Reserved
	binary.BigEndian.PutUint32(buf[8:12], h.PayloadLen)
	copy(buf[12:40], h.SessionID[:])
	return buf
}

// Unmarshal 从字节流解析 Header
func Unmarshal(data []byte) (*Header, error) {
	if len(data) < HeaderSize {
		return nil, errors.New("header too short")
	}
	h := &Header{}
	h.Magic = binary.BigEndian.Uint32(data[0:4])
	if h.Magic != MagicNumber {
		return nil, errors.New("invalid magic number")
	}
	h.Version = binary.BigEndian.Uint16(data[4:6])
	h.Cmd = data[6]
	h.Reserved = data[7]
	h.PayloadLen = binary.BigEndian.Uint32(data[8:12])
	copy(h.SessionID[:], data[12:40])
	return h, nil
}
