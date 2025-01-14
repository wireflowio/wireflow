package conn

import (
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"linkany/pkg/iface"
	"linkany/pkg/internal"
	"sync"
)

type ProberManager struct {
	lock         sync.Mutex
	probers      map[string]*Prober
	wgLock       sync.Mutex
	isForceRelay bool
	wgConfiger   iface.WGConfigure
	relayer      internal.Relay
}

func (pm *ProberManager) AddProber(key wgtypes.Key, prober *Prober) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.probers[key.String()] = prober
}

func (pm *ProberManager) GetProber(key wgtypes.Key) *Prober {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	return pm.probers[key.String()]
}

func NewProberManager(isForceRelay bool) *ProberManager {
	return &ProberManager{
		probers:      make(map[string]*Prober),
		isForceRelay: isForceRelay,
	}
}

func (pm *ProberManager) RemoveProber(key wgtypes.Key) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	delete(pm.probers, key.String())
}

func (pm *ProberManager) SetWgConfiger(wgConfiger iface.WGConfigure) {
	pm.wgLock.Lock()
	defer pm.wgLock.Unlock()
	pm.wgConfiger = wgConfiger
}

func (pm *ProberManager) GetWgConfiger() iface.WGConfigure {
	pm.wgLock.Lock()
	defer pm.wgLock.Unlock()
	return pm.wgConfiger
}

func (pm *ProberManager) IsForceRelay() bool {
	return pm.isForceRelay
}

func (pm *ProberManager) SetRelayer(relayer internal.Relay) {
	pm.wgLock.Lock()
	defer pm.wgLock.Unlock()
	pm.relayer = relayer
}
