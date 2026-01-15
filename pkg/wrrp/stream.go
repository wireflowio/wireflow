package wrrp

import "net"

// Stream abstract the exact transport protocol
type Stream interface {
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
	Close() error
	RemoteAddr() net.Addr
}

type Session struct {
	ID     string
	Stream Stream
	Type   string // TCP / QUIC / KCP
}
