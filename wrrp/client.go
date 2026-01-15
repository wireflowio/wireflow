package wrrp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
	"wireflow/pkg/wrrp"

	"golang.zx2c4.com/wireguard/conn"
)

type WRRPClient struct {
	mu        sync.Mutex
	SessionID [28]byte
	ServerURL string
	Conn      net.Conn
	Reader    *bufio.Reader
}

func NewClient(id [28]byte, url string) *WRRPClient {
	return &WRRPClient{
		SessionID: id,
		ServerURL: url,
	}
}

func (c *WRRPClient) Dial(mode string) error {
	if mode == "tcp" {
		return c.Connect() // 走刚才写的 HTTP Hijack
	} else if mode == "quic" {
		// 将 c.Conn 替换为 quic.Stream
		// 直接发送 Register，跳过 HTTP 握手
	}
	return nil
}

func (c *WRRPClient) Connect() error {
	// 1. 建立 TCP 连接 (如果是 https 则需要 tls.Dial)
	// 这里简化为 http 端口，实际生产建议用 443
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		return err
	}

	// 2. 手动构造 HTTP Upgrade 请求
	// 注意：不能直接用 http.Get，因为我们需要拿回底层的 conn
	req, _ := http.NewRequest("GET", "/wrrp/v1/upgrade", nil)
	req.Header.Set("Upgrade", "wrrp")
	req.Header.Set("Connection", "Upgrade")
	//req.Header.Set("X-WRRP-ID", string(c.SessionID[:]))
	req.Header.Set("Host", "127.0.0.1:8080") // 必须

	if err := req.Write(conn); err != nil {
		return err
	}

	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, req)
	if err != nil || resp.StatusCode != http.StatusSwitchingProtocols {
		return fmt.Errorf("upgrade failed: %v", err)
	}

	// 4. 接管连接
	c.Conn = conn
	c.Reader = reader

	// 5. 立即发送 WRRP 注册报文 (Register)
	return c.register()
}

func (c *WRRPClient) register() error {
	header := &wrrp.Header{
		Magic:      wrrp.MagicNumber,
		Version:    1,
		Cmd:        wrrp.Register,
		PayloadLen: 0,
		SessionID:  c.SessionID,
	}

	_, err := c.Conn.Write(header.Marshal())
	return err
}

// Send 向指定的目标 Peer 发送数据
func (c *WRRPClient) Send(targetID [28]byte, data []byte) error {
	header := &wrrp.Header{
		Magic:      wrrp.MagicNumber,
		Version:    1,
		Cmd:        wrrp.Forward,
		PayloadLen: uint32(len(data)),
		SessionID:  targetID, // 这里填目标 ID
	}

	// 发送 Header + Payload
	if _, err := c.Conn.Write(header.Marshal()); err != nil {
		return err
	}
	_, err := c.Conn.Write(data)
	return err
}

var payloadPool = sync.Pool{
	New: func() interface{} {
		// 申请一个足够大的缓冲区（比如符合 MTU 的 1600 字节）
		return make([]byte, 2048)
	},
}

// HandleFrame 开始循环监听来自 Server 的转发数据
func (c *WRRPClient) HandleFrame(handler func(header *wrrp.Header, payload []byte) (int, conn.Endpoint, error)) {
	for {
		headBuf := make([]byte, wrrp.HeaderSize)
		if _, err := io.ReadFull(c.Reader, headBuf); err != nil {
			fmt.Println("Connection closed by server")
			return
		}

		header, _ := wrrp.Unmarshal(headBuf)
		switch header.Cmd {
		case wrrp.Forward:
			buf := payloadPool.Get().([]byte) // 拿出一块现成的内存
			defer payloadPool.Put(buf)        // 函数结束时还回去

			// 只读取 header 指定的长度
			data := buf[:header.PayloadLen]
			if _, err := io.ReadFull(c.Reader, data); err != nil {
				return
			}
			handler(header, data)
		}

	}
}

func (c *WRRPClient) startKeepAlive(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 构造一个 Ping 包
			header := &wrrp.Header{
				Magic:      wrrp.MagicNumber,
				Version:    1,
				Cmd:        wrrp.Ping,
				PayloadLen: 0,
				SessionID:  c.SessionID,
			}

			c.mu.Lock() // 建议给 Conn 加锁，防止与数据发送冲突
			_, err := c.Conn.Write(header.Marshal())
			c.mu.Unlock()

			if err != nil {
				fmt.Printf("[WRRP] KeepAlive failed: %v\n", err)
				return
			}
		}
	}
}
