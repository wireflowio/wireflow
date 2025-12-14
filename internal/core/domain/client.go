package domain

import (
	"context"
	"wireflow/internal/grpc"
)

// IClient is the interface for managing WireGuard devices.
type IClient interface {
	// Start the engine
	Start() error

	// Stop the engine
	Stop() error

	// GetDeviceConfiger  // Get the WireGuard configuration manager
	GetDeviceConfiger() Configurer

	Configure(conf *DeviceConfig) error

	// AddPeer adds a peer to the WireGuard device, add peer from contrl client, then will start connect to peer
	AddPeer(peer *Peer) error

	// RemovePeer removes a peer from the WireGuard device
	RemovePeer(peer *Peer) error

	RemoveAllPeers()
}

// IKeyManager manage the device keys
type IKeyManager interface {
	// UpdateKey updates the private key used for encryption.
	UpdateKey(privateKey string)
	// GetKey retrieves the current private key.
	GetKey() string
	// GetPublicKey retrieves the public key derived from the current private key.
	GetPublicKey() string
}

type IPeerManager interface {
	AddPeer(key string, peer *Peer)
	GetPeer(key string) *Peer
	RemovePeer(key string)
}

type IManagementClient interface {
	GetNetMap() (*Message, error)
	Register(ctx context.Context, appId string) (*Peer, error)
	AddPeer(p *Peer) error
	Watch(ctx context.Context, fn func(message *Message) error) error
	Keepalive(ctx context.Context) error
}

type IDRPClient interface {
	HandleMessage(ctx context.Context, outBoundQueue chan *grpc.DrpMessage, receive func(ctx context.Context, msg *grpc.DrpMessage) error) error
	Close() error
}
