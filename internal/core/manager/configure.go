package manager

import (
	"wireflow/internal/core/domain"

	wg "golang.zx2c4.com/wireguard/device"
)

type defaultConfiger struct {
	device       *wg.Device
	address      string
	ifaceName    string
	peersManager domain.IPeerManager
}

func (c *defaultConfiger) GetAddress() string {
	return c.address
}

func (c *defaultConfiger) GetIfaceName() string {
	return c.ifaceName
}

func (c *defaultConfiger) GetPeersManager() domain.IPeerManager {
	return c.peersManager
}

type Params struct {
	Device       *wg.Device
	IfaceName    string
	Address      string
	PeersManager domain.IPeerManager
}

func (c *defaultConfiger) Configure() error {
	return nil
}

func (c *defaultConfiger) ConfigSet(conf *domain.DeviceConfig) error {
	return nil
}

func (c *defaultConfiger) AddPeer(peer *domain.SetPeer) error {
	return c.device.IpcSet(peer.String())
}

func (c *defaultConfiger) RemovePeer(peer *domain.SetPeer) error {
	return c.device.IpcSet(peer.String())
}

func (c *defaultConfiger) RemoveAllPeers() {
	c.device.RemoveAllPeers()
}

func NewConfigurer(config *Params) domain.Configurer {
	return &defaultConfiger{
		device:       config.Device,
		address:      config.Address,
		ifaceName:    config.IfaceName,
		peersManager: config.PeersManager,
	}
}
