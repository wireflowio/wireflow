// Copyright 2025 The Wireflow Authors, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package agent implements the Wireflow data-plane node.
// It wraps a WireGuard device and handles peer discovery, NAT traversal,
// and network provisioning on behalf of the local host.
package node

import (
	"context"
	"fmt"
	"net"
	"strings"
	"wireflow/internal"
	"wireflow/internal/config"
	"wireflow/internal/infra"
	"wireflow/internal/log"
	ctrclient "wireflow/management/client"
	"wireflow/management/nats"
	"wireflow/management/transport"
	"wireflow/pkg/utils"
	"wireflow/wrrper"

	wg "golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var (
	_ infra.NodeInterface = (*Node)(nil)
)

// Node is the Wireflow data-plane node. It owns the WireGuard device and
// coordinates peer discovery, ICE/WRRP hole-punching, and OS network
// provisioning (routes, iptables rules, WireGuard peer config).
type Node struct {
	logger      *log.Logger
	Name        string
	iface       *wg.Device
	bind        *infra.DefaultBind
	provisioner infra.Provisioner
	natsService infra.SignalService

	// GetNetworkMap is set externally after NewAgent returns and before Start
	// is called. It fetches the current network topology from the control plane.
	GetNetworkMap func() (*infra.Message, error)
	ctrClient    *ctrclient.Client
	probeFactory *transport.ProbeFactory

	manager struct {
		keyManager  infra.KeyManager
		turnManager *internal.TurnManager
		peerManager *infra.PeerManager
	}

	current *infra.Peer

	token          string
	callback       func(message *infra.Message) error // nolint
	messageHandler Handler

	DeviceManager *DeviceManager
}

// NodeConfig holds the startup parameters for NewNode.
type NodeConfig struct {
	Logger        *log.Logger
	Port          int
	InterfaceName string
	ForceRelay    bool
	ShowLog       bool
	Token         string
	Flags         *config.Config
}

// NewAgent constructs and wires a fully operational Agent instance.
//
// Initialization is split into three strictly ordered phases:
//
// Phase 1 — Network foundation (no business dependencies)
//
//	TUN device → UDP sockets → ICE UDP Mux → NATS signal service
//
// Phase 2 — Identity and signaling (depends on phase 1)
//
//	Register with control plane → derive PrivateKey → build KeyManager/PeerIdentity
//	→ create ProbeFactory (Provisioner is nil at this point, wired in phase 3)
//	→ subscribe NATS topic → wire ControlClient → optional WRRP relay client
//
// Phase 3 — WireGuard data plane (depends on phase 2)
//
//	DefaultBind → WireGuard Device → Provisioner → MessageHandler
//	→ wire ProbeFactory with the now-available Provisioner and MessageHandler
//
// ProbeFactory and ControlClient use two-phase initialization: New() creates
// them with partial dependencies, and Configure() injects the remaining ones
// once they are available in phase 3. This breaks the otherwise circular
// dependency: ProbeFactory ↔ Provisioner ↔ WireGuard Device.
func NewNode(ctx context.Context, cfg *NodeConfig) (*Node, error) {
	var (
		iface      tun.Device
		err        error
		agent      *Node
		v4conn     *net.UDPConn
		v6conn     *net.UDPConn
		wrrp       *wrrper.WRRPClient
		privateKey wgtypes.Key
	)

	// ── Phase 1: Network foundation ──────────────────────────────────────────

	agent = new(Node)
	agent.manager.peerManager = infra.NewPeerManager()
	agent.logger = cfg.Logger
	agent.manager.turnManager = new(internal.TurnManager)

	// TUN device: the OS virtual NIC that serves as WireGuard's L3 ingress/egress.
	agent.Name, iface, err = infra.CreateTUN(infra.DefaultMTU, cfg.Logger)
	if err != nil {
		return nil, err
	}

	// UDP sockets: ICE candidate gathering and WireGuard encapsulated packets
	// share the same port (default 51820) via the ICE UDP Mux below.
	if v4conn, _, err = infra.ListenUDP("udp4", uint16(cfg.Port)); err != nil {
		return nil, err
	}

	if v6conn, _, err = infra.ListenUDP("udp6", uint16(cfg.Port)); err != nil {
		return nil, err
	}

	// ICE UDP Mux: multiplexes a single UDP port across multiple concurrent ICE
	// agents, avoiding the need for a dedicated port per peer connection.
	universalUdpMuxDefault := infra.NewUdpMux(v4conn, cfg.ShowLog)

	// NATS signal service: exchanges ICE signaling messages (SYN/ACK/Offer/Answer)
	// with the control plane and remote peers.
	natsSignalService, err := nats.NewNatsService(ctx, config.Conf.AppId, "client", config.Conf.SignalingURL)
	if err != nil {
		return nil, err
	}
	agent.natsService = natsSignalService

	// ── Phase 2: Identity and signaling ──────────────────────────────────────

	// ControlClient communicates with the management service for registration
	// and network topology retrieval.
	agent.ctrClient, err = ctrclient.NewClient(natsSignalService)
	if err != nil {
		return nil, err
	}

	// Register announces this node to the control plane and receives back the
	// assigned WireGuard private key, allocated IP, and WRRP relay URL.
	agent.current, err = agent.ctrClient.Register(ctx, cfg.Token, agent.Name)
	if err != nil {
		return nil, err
	}

	privateKey, err = utils.ParseKey(agent.current.PrivateKey)
	if err != nil {
		return nil, err
	}
	// KeyManager holds the WireGuard private key and exposes it to the Bind
	// layer so it can perform AEAD peer matching during the handshake.
	agent.manager.keyManager = infra.NewKeyManager(privateKey)

	// PeerIdentity is this node's unique signaling identity: AppID + PublicKey.
	// It is used as the ICE tiebreaker to deterministically elect the ICE
	// initiator when two peers attempt to connect simultaneously.
	localIdentity := infra.NewPeerIdentity(agent.current.AppID, privateKey.PublicKey())

	// Register this node in the PeerManager so hole-punching logic can look up
	// local peer info during ICE negotiation.
	agent.manager.peerManager.AddPeer(agent.current.AppID, agent.current)

	// ProbeFactory manages the lifecycle of per-peer connection probes (ICE
	// hole-punching, WRRP relay fallback). Provisioner is nil here and will be
	// injected in phase 3 via Configure() once the WireGuard device exists.
	agent.probeFactory = transport.NewProbeFactory(&transport.ProbeFactoryConfig{
		LocalId:                localIdentity,
		Signal:                 natsSignalService,
		PeerManager:            agent.manager.peerManager,
		UniversalUdpMuxDefault: universalUdpMuxDefault,
		Provisioner:            agent.provisioner, // nil; wired in phase 3
		ShowLog:                cfg.ShowLog,
	})

	// Subscribe to this node's NATS signaling subject. All incoming ICE and
	// WRRP signal packets are routed to probeFactory.Handle for dispatch.
	if err = natsSignalService.Subscribe(fmt.Sprintf("%s.%s", "wireflow.signals.peers", localIdentity), agent.probeFactory.Handle); err != nil {
		return nil, err
	}

	// Wire ControlClient: inject KeyManager and ProbeFactory so that when the
	// control plane pushes a PeersAdded event, ControlClient can trigger
	// hole-punching immediately.
	agent.ctrClient.Configure(
		ctrclient.WithSignalHandler(natsSignalService),
		ctrclient.WithKeyManager(agent.manager.keyManager),
		ctrclient.WithProbeFactory(agent.probeFactory))

	// WRRP is an optional relay channel used as a fallback when ICE traversal
	// fails (e.g. symmetric NAT on both sides).
	if cfg.Flags.EnableWrrp {
		wrrpUrl := cfg.Flags.WrrperURL
		if wrrpUrl == "" {
			wrrpUrl = agent.current.WrrpUrl
		}

		if wrrpUrl != "" {
			wrrp, err = wrrper.NewWrrpClient(localIdentity.ID(), wrrpUrl)
			if err != nil {
				return nil, err
			}
			// Route inbound WRRP signal packets through the same probeFactory handler.
			wrrp.Configure(wrrper.WithOnMessage(agent.probeFactory.Handle))
		}
	}

	// ── Phase 3: WireGuard data plane ────────────────────────────────────────

	// DefaultBind is WireGuard's UDP binding layer. It routes outbound encrypted
	// packets to the correct transport channel (ICE direct path or WRRP relay)
	// and uses KeyManager to match inbound packets to the right WireGuard peer
	// during the handshake.
	agent.bind = infra.NewBind(&infra.BindConfig{
		Logger:          cfg.Logger,
		UniversalUDPMux: universalUdpMuxDefault,
		V4Conn:          v4conn,
		V6Conn:          v6conn,
		WrrpClient:      wrrp,
		KeyManager:      agent.manager.keyManager,
	})

	wgLogLevel := wg.LogLevelError
	if cfg.ShowLog {
		wgLogLevel = wg.LogLevelVerbose
	}
	// WireGuard Device: the data-plane core. It encrypts/decrypts packets and
	// hands them off to the TUN device or Bind layer as appropriate.
	agent.iface = wg.NewDevice(iface, agent.bind, wg.NewLogger(wgLogLevel, fmt.Sprintf("(%s) ", cfg.InterfaceName)))

	// Provisioner abstracts all OS network-stack mutations: IP address assignment,
	// routing table entries, iptables rules, and WireGuard peer configuration.
	// It must be created after the WireGuard device because it holds a reference to it.
	agent.provisioner = infra.NewProvisioner(infra.NewRouteProvisioner(cfg.Logger),
		infra.NewRuleProvisioner(cfg.Logger, agent.Name), &infra.Params{
			Device:    agent.iface,
			IfaceName: agent.Name,
		})

	// MessageHandler processes topology change events pushed by the control plane
	// (peers added/removed, configuration updates) and applies them via Provisioner.
	agent.messageHandler = NewMessageHandler(agent, log.GetLogger("event-handler"), agent.provisioner)

	// Wire ProbeFactory: inject the Provisioner and MessageHandler that were
	// unavailable during phase 2. After a successful hole-punch, ProbeFactory
	// calls onMessage so MessageHandler can install the new direct route.
	agent.probeFactory.Configure(
		transport.WithOnMessage(agent.messageHandler.HandleEvent),
		transport.WithWrrp(wrrp),
		transport.WithProvisioner(agent.provisioner),
	)

	agent.DeviceManager = NewDeviceManager(log.GetLogger("device-manager"), agent.iface, make(chan struct{}))
	agent.token = cfg.Token

	// Re-register and re-apply the network map whenever NATS reconnects.
	// This covers the case where wireflow-aio restarts and loses all agent state.
	// The handler reads GetNetworkMap at call time (not at setup time), so it
	// works even though GetNetworkMap is assigned externally after NewAgent returns.
	natsSignalService.SetReconnectedHandler(func() {
		ctx := context.Background()
		peer, err := agent.ctrClient.Register(ctx, agent.token, agent.Name)
		if err != nil {
			agent.logger.Error("NATS reconnect: re-register failed", err)
			return
		}
		agent.current = peer

		if agent.GetNetworkMap == nil {
			return
		}
		remoteCfg, err := agent.GetNetworkMap()
		if err != nil {
			agent.logger.Error("NATS reconnect: re-fetch network map failed", err)
			return
		}
		if err = agent.messageHandler.ApplyFullConfig(ctx, remoteCfg); err != nil {
			agent.logger.Error("NATS reconnect: re-apply config failed", err)
		}
	})

	return agent, err
}

// Start brings up the WireGuard data plane and applies the initial network
// configuration fetched from the control plane.
//
// Call order:
//  1. Bring the WireGuard device up (begin sending/receiving UDP packets).
//  2. Write the WireGuard private key and interface settings to the OS.
//  3. Fetch the current network topology via GetNetworkMap.
//  4. Add all remote peers to WireGuard and establish initial routes.
//
// Must be called after NewAgent returns and after GetNetworkMap has been set.
func (c *Node) Start(ctx context.Context) error {
	if err := c.iface.Up(); err != nil {
		return err
	}

	if err := c.provisioner.SetupInterface(&infra.DeviceConfig{
		PrivateKey: c.current.PrivateKey,
	}); err != nil {
		return err
	}

	remoteCfg, err := c.GetNetworkMap()
	if err != nil {
		return err
	}

	return c.messageHandler.ApplyFullConfig(ctx, remoteCfg)
}

// Stop gracefully shuts down the Agent. It drains the NATS connection first
// so the server immediately removes this node's subscriptions, preventing
// "no responders" errors on peer reconnect attempts. Then it closes the
// WireGuard device, releasing the TUN interface and UDP sockets.
func (c *Node) Stop() error {
	if c.natsService != nil {
		if err := c.natsService.Close(); err != nil {
			c.logger.Warn("nats drain failed", "err", err)
		}
	}
	c.iface.Close()
	return nil
}

// SetConfig updates the WireGuard device configuration via the kernel IPC
// interface. It reads the current config first and skips the write if nothing
// has changed, avoiding unnecessary syscalls.
func (c *Node) SetConfig(conf *infra.DeviceConf) error {
	nowConf, err := c.iface.IpcGet()
	if err != nil {
		return err
	}

	if conf.String() == nowConf {
		c.logger.Debug("config is same, no need to update", "conf", conf)
		return nil
	}

	reader := strings.NewReader(conf.String())

	return c.iface.IpcSetOperation(reader)
}

// nolint:unused
func (c *Node) close() {
	c.logger.Debug("deviceManager closed")
}

// AddPeer registers a remote peer with the local node. It first updates the
// in-memory PeerManager (used by hole-punching probes to look up peer info),
// then writes the WireGuard peer configuration via ControlClient. If the peer
// is this node itself (matching public key), the WireGuard write is skipped.
func (c *Node) AddPeer(peer *infra.Peer) error {
	c.manager.peerManager.AddPeer(peer.AppID, peer)
	if peer.PublicKey == c.current.PublicKey {
		return nil
	}
	return c.ctrClient.AddPeer(peer)
}

//func (c *Node) Configure(peerId string) error {
//	//conf *infra.DeviceConfig
//	peer := c.manager.peerManager.GetPeer(peerId.ToUint64())
//	if peer == nil {
//		return errors.New("peer not found")
//	}
//
//	conf := &infra.DeviceConfig{
//		PrivateKey: peer.PrivateKey,
//	}
//	return c.provisioner.SetupInterface(conf)
//}

// RemovePeer evicts a remote peer from the local node. It closes and removes
// the associated Probe (stopping reconnection attempts), then deletes the
// WireGuard peer configuration. A new Probe will be created automatically
// when the control plane pushes a PeersAdded event for this peer again.
func (c *Node) RemovePeer(peer *infra.Peer) error {
	c.probeFactory.Remove(peer.AppID)
	return c.provisioner.RemovePeer(&infra.SetPeer{
		Remove:    true,
		PublicKey: peer.PublicKey,
	})
}

func (c *Node) RemoveAllPeers() {
	c.provisioner.RemoveAllPeers()
}

func (c *Node) GetDeviceName() string {
	return c.Name
}

func (c *Node) GetPeerManager() *infra.PeerManager {
	return c.manager.peerManager
}
