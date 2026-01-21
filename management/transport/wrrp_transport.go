package transport

import (
	"context"
	"net"
	"wireflow/internal/grpc"
	"wireflow/internal/infra"
)

var (
	_ infra.Transport = (*wrrpTransport)(nil)
)

type wrrpTransport struct {
	localId   string
	sessionId [28]byte
}

func (w wrrpTransport) Prepare() error {
	//TODO implement me
	panic("implement me")
}

func (w wrrpTransport) HandleOffer(ctx context.Context, peerId string, packet *grpc.SignalPacket) error {
	//TODO implement me
	panic("implement me")
}

func (w wrrpTransport) Start(ctx context.Context, peerId string) error {
	//TODO implement me
	panic("implement me")
}

func (w wrrpTransport) RawConn() (net.Conn, error) {
	//TODO implement me
	panic("implement me")
}

func (w wrrpTransport) Close() error {
	//TODO implement me
	panic("implement me")
}
