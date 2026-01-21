package transport

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"time"
	"wireflow/internal/grpc"
	"wireflow/internal/infra"
	"wireflow/internal/log"

	"github.com/pion/logging"
	"github.com/wireflowio/ice"
	"google.golang.org/protobuf/proto"
)

var (
	_ infra.Dialer = (*iceDialer)(nil)
)

type iceDialer struct {
	su          sync.Mutex
	mu          sync.Mutex
	log         *log.Logger
	localId     string
	sender      func(ctx context.Context, peerId string, data []byte) error
	onClose     func(peerId string)
	provisioner infra.Provisioner
	remoteId    string
	agent       *AgentWrapper
	closeOnce   sync.Once
	ackClose    sync.Once
	showLog     bool
	peers       *infra.PeerManager

	// offerReady start Dial() after receiving offer
	offerReady chan struct{}
	cancel     context.CancelFunc
	ackChan    chan struct{}

	universalUdpMuxDefault *ice.UniversalUDPMuxDefault
}

type ICEDialerConfig struct {
	Sender                  func(ctx context.Context, peerId string, data []byte) error
	RemoteId                string
	LocalId                 string
	OnClose                 func(peerId string)
	UniversalUdpMuxDefault  *ice.UniversalUDPMuxDefault
	Configurer              infra.Provisioner
	PeerManager             *infra.PeerManager
	ShowLog                 bool
	OnConnectionStateChange func(state ice.ConnectionState)
}

func (i *iceDialer) HandleSignal(ctx context.Context, remoteId string, packet *grpc.SignalPacket) error {
	switch packet.Type {
	case grpc.PacketType_HANDSHAKE_ACK:
		// cancel send syn
		i.cancel()
		// start send offer
		return i.agent.GatherCandidates()
	case grpc.PacketType_HANDSHAKE_SYN:
		// send ack to remote
		if err := i.sendPacket(ctx, i.remoteId, grpc.PacketType_HANDSHAKE_ACK, nil); err != nil {
			return err
		}

		// start send offer (locaId < remoteId)
		return i.agent.GatherCandidates()
	case grpc.PacketType_OFFER, grpc.PacketType_ANSWER:
		i.log.Info("receive offer", "remoteId", remoteId)
		offer := packet.GetOffer() //第一次接收
		if !i.agent.IsCredentialsInited.Load() {
			i.agent.RUfrag = offer.Ufrag
			i.agent.RPwd = offer.Pwd
			i.agent.RTieBreaker = offer.TieBreaker
			i.agent.IsCredentialsInited.Store(true)
		}

		candidate, err := ice.UnmarshalCandidate(offer.Candidate)
		if err != nil {
			return err
		}

		if err = i.agent.AddRemoteCandidate(candidate); err != nil {
			return err
		}

		i.log.Info("add remote candidate", "candidate", candidate)
		i.closeOnce.Do(func() {
			close(i.offerReady)
		})
	}
	return nil
}

func NewIceDialer(cfg *ICEDialerConfig) infra.Dialer {
	return &iceDialer{
		log:                    log.GetLogger("ice-dialer"),
		sender:                 cfg.Sender,
		onClose:                cfg.OnClose,
		localId:                cfg.LocalId,
		remoteId:               cfg.RemoteId,
		universalUdpMuxDefault: cfg.UniversalUdpMuxDefault,
		showLog:                cfg.ShowLog,
		peers:                  cfg.PeerManager,
		offerReady:             make(chan struct{}),
	}
}

// Prepare prepare to send offer, send handshake packet first to remote when localId > remoteId.
func (i *iceDialer) Prepare(ctx context.Context, remoteId string) error {
	// init agent
	if i.agent == nil {
		agent, err := i.getAgent(remoteId)
		if err != nil {
			panic(err)
		}
		i.agent = agent
	}

	i.log.Info("prepare ice", "localId", i.localId, "remoteId", remoteId, "shouldSync", i.localId > remoteId)
	// only send syn when localId > remoteId
	if i.localId < remoteId {
		i.log.Info("localId < remoteId, ignore prepare")
		return nil
	}

	// send syn
	go func() {
		// send syn
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		// safe
		i.mu.Lock()
		i.cancel = cancel
		i.mu.Unlock()
		for {
			select {
			case <-ctx.Done():
				i.log.Warn("send syn canceled", "err", ctx.Err())
				return
			case <-ticker.C:
				i.log.Info("send syn")
				err := i.sendPacket(ctx, remoteId, grpc.PacketType_HANDSHAKE_SYN, nil)
				if err != nil {
					i.log.Error("send syn failed", err)
				}
			}
		}
	}()

	return nil
}

func (i *iceDialer) Dial(ctx context.Context) (net.Conn, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-i.offerReady:
		i.log.Info("start dial")
		if i.agent.GetTieBreaker() > i.agent.RTieBreaker {
			return i.agent.Dial(ctx, i.agent.RUfrag, i.agent.RPwd)
		} else {
			return i.agent.Accept(ctx, i.agent.RUfrag, i.agent.RPwd)
		}
	}
}

func (i *iceDialer) Type() infra.DialerType {
	return infra.ICE_DIALER
}

func (i *iceDialer) getAgent(remoteId string) (*AgentWrapper, error) {
	f := logging.NewDefaultLoggerFactory()
	f.DefaultLogLevel = logging.LogLevelDebug
	if i.showLog {
	} else {
		f.DefaultLogLevel = logging.LogLevelError
	}
	// 创建新 Agent
	iceAgent, err := ice.NewAgent(&ice.AgentConfig{
		UDPMux:         i.universalUdpMuxDefault.UDPMuxDefault,
		UDPMuxSrflx:    i.universalUdpMuxDefault,
		NetworkTypes:   []ice.NetworkType{ice.NetworkTypeUDP4},
		Urls:           []*ice.URL{{Scheme: ice.SchemeTypeSTUN, Host: "stun.wireflow.run", Port: 3478}},
		Tiebreaker:     uint64(ice.NewTieBreaker()),
		LoggerFactory:  f,
		CandidateTypes: []ice.CandidateType{ice.CandidateTypeHost, ice.CandidateTypeServerReflexive},
	})

	//iceAgent, err := ice.NewAgent(&ice.AgentConfig{
	//	NetworkTypes:  []ice.NetworkType{ice.NetworkTypeUDP4},
	//	LoggerFactory: f,
	//	Tiebreaker:    uint64(ice.NewTieBreaker()),
	//})

	var agent *AgentWrapper
	if err == nil {
		agent = &AgentWrapper{
			Agent: iceAgent,
		}
		// 绑定状态监听，成功后更新 WireGuard
		agent.OnConnectionStateChange(func(s ice.ConnectionState) {
			i.log.Info("ice state changed", "state", s)
			if s == ice.ConnectionStateConnected {

			}

			if s == ice.ConnectionStateDisconnected || s == ice.ConnectionStateFailed {
				i.close()
			}
		})
	}

	if err = agent.OnCandidate(func(candidate ice.Candidate) {
		if candidate == nil {
			return
		}

		if err = i.sendPacket(context.TODO(), remoteId, grpc.PacketType_OFFER, candidate); err != nil {
			i.log.Error("Send candidate", err)
		}

		i.log.Info("Sending candidate", "remoteId", remoteId, "candidate", candidate)
	}); err != nil {
		return nil, err
	}

	return agent, err
}

func (i *iceDialer) sendPacket(ctx context.Context, remoteId string, packetType grpc.PacketType, candidate ice.Candidate) error {
	p := &grpc.SignalPacket{
		Type:     packetType,
		SenderId: i.localId,
	}

	switch packetType {
	case grpc.PacketType_HANDSHAKE_SYN, grpc.PacketType_HANDSHAKE_ACK:
		p.Payload = &grpc.SignalPacket_Handshake{
			Handshake: &grpc.Handshake{
				Timestamp: time.Now().Unix(),
			},
		}
	case grpc.PacketType_OFFER:
		agent := i.agent
		current := i.peers.GetPeer(i.localId)
		currentData, err := json.Marshal(current)
		if err != nil {
			return err
		}
		ufrag, pwd, err := agent.GetLocalUserCredentials()
		if err != nil {
			return err
		}
		p.Payload = &grpc.SignalPacket_Offer{
			Offer: &grpc.Offer{
				Ufrag:      ufrag,
				Pwd:        pwd,
				TieBreaker: agent.GetTieBreaker(),
				Candidate:  candidate.Marshal(),
				Current:    currentData,
			},
		}
	}
	data, err := proto.Marshal(p)
	if err != nil {
		return err
	}
	return i.sender(ctx, remoteId, data)

}

func (i *iceDialer) close() error {
	i.log.Info("closing ice", "remoteId", i.remoteId)
	i.closeOnce.Do(func() {
		if err := i.agent.Close(); err != nil {
			i.log.Error("close agent", err)
		}

		if i.onClose != nil {
			i.onClose(i.remoteId)
		}

		//remove peer
		//i.Remove(t.remoteId, "")
	})

	return nil
}
