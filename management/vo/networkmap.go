package vo

import (
	"github.com/alatticeio/lattice/internal/infra"
)

type NetworkMap struct {
	UserId  string
	Current *PeerVo
	Nodes   []*infra.Peer
}
