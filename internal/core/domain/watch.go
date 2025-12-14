package domain

import (
	"sync"
)

type NodeChannel struct {
	nu        sync.Mutex
	NetworkId []string
	Channel   chan *Message // key: clientId, value: Channel
}

func (n *NodeChannel) GetChannel() chan *Message {
	n.nu.Lock()
	defer n.nu.Unlock()
	if n.Channel == nil {
		n.Channel = make(chan *Message, 1000) // buffered Channel
	}
	return n.Channel
}

// IWatchManager used to manage all the channels of connected peers
type IWatchManager interface {
	GetChannel(clientId string) *NodeChannel
	Remove(clientID string)
	Send(clientId string, msg *Message) error
}
