package infra

import (
	"context"
	"net"
	"wireflow/internal/grpc"
)

type Dialer interface {
	// Prepare prepare offer will send to remoteId.
	Prepare(ctx context.Context, remoteId string) error

	// HandleSignal using for handle si
	HandleSignal(ctx context.Context, remoteId string, packet *grpc.SignalPacket) error

	// Dial dial remoteId when receive offer
	Dial(ctx context.Context) (net.Conn, error)

	// Type return the dialer type
	Type() DialerType
}

type DialerType string

const (
	ICE_DIALER  DialerType = "ICE_DIALER"
	WRRP_DIALER DialerType = "WRRP_DIALER"
)
