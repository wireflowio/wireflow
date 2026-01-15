package infra

import (
	"net/netip"
)

// 1. 自定义一个极简的 Endpoint
type WRRPEndpoint struct {
	SessionID string // 使用 WRRP 的 SessionID 作为唯一标识
}

func (e *WRRPEndpoint) Clear()              {}
func (e *WRRPEndpoint) DstToString() string { return "wrrp:" + e.SessionID }
func (e *WRRPEndpoint) DstToBytes() []byte  { return []byte(e.SessionID) }
func (e *WRRPEndpoint) DstIP() netip.Addr   { return netip.Addr{} } // WRRP 隧道不需要真实 IP
func (e *WRRPEndpoint) SrcIP() netip.Addr   { return netip.Addr{} }
func (e *WRRPEndpoint) SrcToString() string { return "" }

//// 2. 在 Receive 方法中使用它
//func (b *WRRPBind) Receive(buff []byte) (int, conn.Endpoint, error) {
//	// 从 WRRP 隧道读取数据
//	data, ok := <-b.inboundChan
//	if !ok {
//		return 0, nil, io.EOF
//	}
//
//	n := copy(buff, data)
//
//	// 返回我们伪造的 Endpoint
//	// 这样 WireGuard 内部就会记住：这个 Peer 对应的是这个 WRRPEndpoint
//	return n, &WRRPEndpoint{SessionID: b.targetSessionID}, nil
//}
