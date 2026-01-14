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

//go:build !windows

package agent

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"wireflow/internal"
	"wireflow/internal/config"
	"wireflow/internal/infra"
	"wireflow/internal/log"
	ctrclient "wireflow/management/client"
	mgtclient "wireflow/management/client"
	"wireflow/management/nats"
	"wireflow/management/transport"

	wg "golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
)

var (
	_ infra.AgentInterface = (*Agent)(nil)
)

// Agent act as wireflow data plane, wrappers around wireguard device
type Agent struct {
	logger      *log.Logger
	Name        string
	iface       *wg.Device
	bind        *infra.DefaultBind
	provisioner infra.Provisioner

	GetNetworkMap func() (*infra.Message, error)
	ctrClient     *ctrclient.Client

	manager struct {
		keyManager  infra.KeyManager
		turnManager *internal.TurnManager
		peerManager *infra.PeerManager
	}

	current *infra.Peer

	callback     func(message *infra.Message) error
	eventHandler Handler

	DeviceManager *DeviceManager
}

type AgentConfig struct {
	Logger        *log.Logger
	Port          int
	UdpConn       *net.UDPConn
	InterfaceName string
	client        *mgtclient.Client
	WgLogger      *wg.Logger
	deviceManager *DeviceManager
	TurnServerUrl string
	ForceRelay    bool
	ManagementUrl string
	SignalingUrl  string
	ShowWgLog     bool
	Token         string
}

// NewAgent create a new Agent instance
func NewAgent(ctx context.Context, cfg *AgentConfig) (*Agent, error) {
	var (
		iface  tun.Device
		err    error
		client *Agent
		//turnClient internal.Agent
		v4conn *net.UDPConn
		v6conn *net.UDPConn
	)
	client = new(Agent)
	client.manager.peerManager = infra.NewPeerManager()
	client.logger = cfg.Logger
	client.manager.turnManager = new(internal.TurnManager)
	client.Name, iface, err = infra.CreateTUN(infra.DefaultMTU, cfg.Logger)
	if err != nil {
		return nil, err
	}

	if v4conn, _, err = infra.ListenUDP("udp4", uint16(cfg.Port)); err != nil {
		return nil, err
	}

	if v6conn, _, err = infra.ListenUDP("udp6", uint16(cfg.Port)); err != nil {
		return nil, err
	}

	universalUdpMuxDefault := infra.NewUdpMux(v4conn)

	natsSignalService, err := nats.NewNatsService(ctx, config.GlobalConfig.SignalUrl)
	if err != nil {
		return nil, err
	}

	factory := transport.NewTransportFactory(natsSignalService, universalUdpMuxDefault)

	client.ctrClient, err = ctrclient.NewClient(natsSignalService, factory)
	if err != nil {
		return nil, err
	}

	var privateKey string
	client.current, err = client.ctrClient.Register(context.Background(), cfg.Token, client.Name)
	if err != nil {
		return nil, err
	}

	// write token
	if err = config.WriteConfig("token", client.current.Token); err != nil {
		return nil, err
	}

	privateKey = client.current.PrivateKey
	client.manager.keyManager = infra.NewKeyManager(privateKey)

	factory.Configure(transport.WithPeerManager(client.manager.peerManager), transport.WithKeyManager(client.manager.keyManager))

	localId := client.manager.keyManager.GetPublicKey()
	probeFactory := transport.NewProbeFactory(&transport.ProbeFactoryConfig{
		Factory: factory,
		LocalId: localId,
		Signal:  natsSignalService,
	})

	//subscribe
	natsSignalService.Subscribe(fmt.Sprintf("%s.%s", "wireflow.signals.peers", localId), probeFactory.HandleSignal)

	client.ctrClient.Configure(
		ctrclient.WithSignalHandler(natsSignalService),
		ctrclient.WithKeyManager(client.manager.keyManager),
		ctrclient.WithProbeFactory(probeFactory))

	client.bind = infra.NewBind(&infra.BindConfig{
		Logger:          cfg.Logger,
		UniversalUDPMux: universalUdpMuxDefault,
		V4Conn:          v4conn,
		V6Conn:          v6conn,
		KeyManager:      client.manager.keyManager,
	})

	stunUrl := config.GlobalConfig.StunUrl
	if stunUrl == "" {
		stunUrl = "stun.wireflow.run"
		config.WriteConfig("stun-url", stunUrl)
	}

	client.iface = wg.NewDevice(iface, client.bind, cfg.WgLogger)

	client.provisioner = infra.NewProvisioner(infra.NewRouteProvisioner(cfg.Logger),
		infra.NewRuleProvisioner(cfg.Logger), &infra.Params{
			Device:    client.iface,
			IfaceName: client.Name,
		})
	// init event handler
	client.eventHandler = NewEventHandler(client, log.GetLogger("event-handler"), client.provisioner)
	// set configurer
	factory.Configure(transport.WithProvisioner(client.provisioner))

	probeFactory.Configure(transport.WithOnMessage(client.eventHandler.HandleEvent))

	client.DeviceManager = NewDeviceManager(log.GetLogger("device-manager"), client.iface)
	return client, err
}

// Start will get networkmap
func (c *Agent) Start(ctx context.Context) error {
	// start deviceManager, open udp port
	if err := c.iface.Up(); err != nil {
		return err
	}

	if c.current.Address != nil {
		// 设置Device
		if err := c.provisioner.ApplyIP("add", *c.current.Address, c.provisioner.GetIfaceName()); err != nil {
			return err
		}
	}

	if c.manager.keyManager.GetKey() != "" {
		if err := c.provisioner.SetupInterface(&infra.DeviceConfig{
			PrivateKey: c.current.PrivateKey,
		}); err != nil {
			return err
		}
	}

	// get network map
	remoteCfg, err := c.GetNetworkMap()
	if err != nil {
		return err
	}

	c.eventHandler.ApplyFullConfig(ctx, remoteCfg)

	return nil
}

func (c *Agent) Stop() error {
	c.iface.Close()
	return nil
}

// SetConfig updates the configuration of the given interface.
func (c *Agent) SetConfig(conf *infra.DeviceConf) error {
	nowConf, err := c.iface.IpcGet()
	if err != nil {
		return err
	}

	if conf.String() == nowConf {
		c.logger.Info("config is same, no need to update", "conf", conf)
		return nil
	}

	reader := strings.NewReader(conf.String())

	return c.iface.IpcSetOperation(reader)
}

func (c *Agent) close() {
	c.logger.Info("deviceManager closed")
}

func (c *Agent) AddPeer(peer *infra.Peer) error {
	c.manager.peerManager.AddPeer(peer.PublicKey, peer)
	if peer.PublicKey == c.current.PublicKey {
		return nil
	}
	return c.ctrClient.AddPeer(peer)
}

func (c *Agent) Configure(peerId string) error {
	//conf *infra.DeviceConfig
	peer := c.manager.peerManager.GetPeer(peerId)
	if peer == nil {
		return errors.New("peer not found")
	}

	conf := &infra.DeviceConfig{
		PrivateKey: peer.PrivateKey,
	}
	return c.provisioner.SetupInterface(conf)
}

func (c *Agent) RemovePeer(peer *infra.Peer) error {
	return c.provisioner.RemovePeer(&infra.SetPeer{
		Remove:    true,
		PublicKey: peer.PublicKey,
	})
}

func (c *Agent) RemoveAllPeers() {
	c.provisioner.RemoveAllPeers()
}

func (c *Agent) GetDeviceName() string {
	return c.Name
}
