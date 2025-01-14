package device

import (
	"golang.zx2c4.com/wireguard/conn"
	wg "golang.zx2c4.com/wireguard/device"
	linkconn "linkany/pkg/conn"
	"linkany/pkg/iface"
)

type Backend struct {
	device       *wg.Device
	bind         conn.Bind
	relayChecker linkconn.RelayChecker
	wgConfiger   iface.WGConfigure
}
