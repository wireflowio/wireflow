package wrrp

import (
	"bufio"
	"crypto/sha256"
	"net"
)

// ReadWriterConn wrapper for missed data when hijack occur， for using Read/Write fn
type ReadWriterConn struct {
	net.Conn
	*bufio.ReadWriter
}

func (c *ReadWriterConn) Read(p []byte) (int, error) {
	return c.ReadWriter.Read(p)
}

func (c *ReadWriterConn) Write(p []byte) (int, error) {
	n, err := c.ReadWriter.Write(p)
	if err != nil {
		return n, err
	}
	// 确保数据立即发出，而不是留在 bufio 的写缓存里
	return n, c.ReadWriter.Flush()
}

func IDFromPublicKey(pubKey []byte) [28]byte {
	// SHA-224 的输出固定为 28 字节
	return sha256.Sum224(pubKey)
}
