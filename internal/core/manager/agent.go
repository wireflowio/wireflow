package manager

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"wireflow/internal/core/domain"
	"wireflow/pkg/log"
	"wireflow/pkg/wferrors"

	"github.com/pion/logging"
	"github.com/pion/randutil"
	"github.com/pion/stun/v3"
	"github.com/wireflowio/ice"
)

var (
	_ domain.IAgent              = (*Agent)(nil)
	_ domain.AgentManagerFactory = (*instance)(nil)
)

const (
	runesAlpha                 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	runesDigit                 = "0123456789"
	runesCandidateIDFoundation = runesAlpha + runesDigit + "+/"

	lenUFrag = 16
	lenPwd   = 32
)

var (
	globalMathRandomGenerator = randutil.NewMathRandomGenerator()
)

type instance struct {
	lock   sync.Locker
	agents map[string]domain.IAgent
}

func NewAgentManager() domain.AgentManagerFactory {
	return &instance{
		agents: make(map[string]domain.IAgent, 1),
	}
}

func (i *instance) Get(pubKey string) (domain.IAgent, error) {
	if agent, ok := i.agents[pubKey]; ok {
		return agent, nil
	}

	return nil, wferrors.ErrAgentNotFound
}

func (i *instance) Remove(pubKey string) error {
	i.lock.Lock()
	defer i.lock.Unlock()

	if agent, ok := i.agents[pubKey]; ok {
		_ = agent.Close()
		delete(i.agents, pubKey)
		return nil
	}

	return wferrors.ErrAgentNotFound
}

func (i *instance) NewUdpMux(conn net.PacketConn) *ice.UniversalUDPMuxDefault {
	loggerFactory := logging.NewDefaultLoggerFactory()
	loggerFactory.DefaultLogLevel = logging.LogLevelDebug

	universalUdpMux := ice.NewUniversalUDPMuxDefault(ice.UniversalUDPMuxParams{
		Logger:  loggerFactory.NewLogger("wrapper"),
		UDPConn: conn,
		Net:     nil,
	})

	return universalUdpMux
}

// Agent represents an ICE agent with its associated local key.
type Agent struct {
	lock            sync.Mutex
	started         atomic.Bool
	logger          *log.Logger
	iceAgent        *ice.Agent
	LocalKey        uint32
	udpMux          *ice.UDPMuxDefault
	universalUdpMux *ice.UniversalUDPMuxDefault
}

// NewAgent creates a new ICE agent, will use to gather candidates
// each peer will create an agent for connection establishment
func NewAgent(params *AgentConfig) (*Agent, error) {
	var (
		err     error
		agent   *ice.Agent
		stunUri []*stun.URI
		uri     *stun.URI
	)

	l := logging.NewDefaultLoggerFactory()
	l.DefaultLogLevel = logging.LogLevelDebug
	if uri, err = stun.ParseURI(fmt.Sprintf("%s:%s", "stun", params.StunUrl)); err != nil {
		return nil, err
	}

	uri.Username = "admin"
	uri.Password = "admin"
	stunUri = append(stunUri, uri)
	f := logging.NewDefaultLoggerFactory()
	f.DefaultLogLevel = logging.LogLevelDebug
	if agent, err = ice.NewAgent(&ice.AgentConfig{
		NetworkTypes:   []ice.NetworkType{ice.NetworkTypeUDP4},
		UDPMux:         params.UniversalUdpMux.UDPMuxDefault,
		UDPMuxSrflx:    params.UniversalUdpMux,
		Tiebreaker:     uint64(ice.NewTieBreaker()),
		Urls:           stunUri,
		LoggerFactory:  f,
		CandidateTypes: []ice.CandidateType{ice.CandidateTypeHost, ice.CandidateTypeServerReflexive},
	}); err != nil {
		return nil, err
	}

	a := &Agent{
		iceAgent:        agent,
		LocalKey:        ice.NewTieBreaker(),
		universalUdpMux: params.UniversalUdpMux,
		logger:          log.NewLogger(log.Loglevel, "agent"),
	}

	a.started.Store(false)
	return a, nil
}

func (agent *Agent) GetStatus() bool {
	if agent.started.Load() {
		return true
	}
	return false
}

func (agent *Agent) GetUniversalUDPMuxDefault() *ice.UniversalUDPMuxDefault {
	return agent.universalUdpMux
}

func (agent *Agent) OnCandidate(fn func(ice.Candidate)) error {
	return agent.iceAgent.OnCandidate(fn)
}

func (agent *Agent) OnConnectionStateChange(fn func(ice.ConnectionState)) error {
	if fn != nil {
		return agent.iceAgent.OnConnectionStateChange(fn)
	}
	return nil
}

func (agent *Agent) AddRemoteCandidate(candidate ice.Candidate) error {
	if agent.iceAgent == nil {
		return nil
	}
	return agent.iceAgent.AddRemoteCandidate(candidate)
}

func (agent *Agent) GatherCandidates() error {
	if agent.iceAgent == nil {
		return nil
	}
	return agent.iceAgent.GatherCandidates()
}

func (agent *Agent) GetLocalCandidates() ([]ice.Candidate, error) {
	if agent.iceAgent == nil {
		return nil, errors.New("ICE agent is not initialized")
	}
	return agent.iceAgent.GetLocalCandidates()
}

func (agent *Agent) GetTieBreaker() uint64 {
	if agent.iceAgent == nil {
		return 0
	}
	return agent.iceAgent.GetTieBreaker()
}

func (agent *Agent) Dial(ctx context.Context, remoteUfrag, remotePwd string) (*ice.Conn, error) {
	if agent.iceAgent == nil {
		return nil, errors.New("ICE agent is not initialized")
	}

	conn, err := agent.iceAgent.Dial(ctx, remoteUfrag, remotePwd)
	if err != nil {
		agent.logger.Errorf("failed to accept ICE connection: %v", err)
		return nil, err
	}
	return conn, nil
}

func (agent *Agent) Accept(ctx context.Context, remoteUfrag, remotePwd string) (*ice.Conn, error) {
	if agent.iceAgent == nil {
		return nil, errors.New("ICE agent is not initialized")
	}

	conn, err := agent.iceAgent.Accept(ctx, remoteUfrag, remotePwd)
	if err != nil {
		agent.logger.Errorf("failed to accept ICE connection: %v", err)
		return nil, err
	}
	return conn, nil
}

func (agent *Agent) Close() error {
	if agent.iceAgent == nil {
		return nil
	}
	if err := agent.iceAgent.Close(); err != nil {
		agent.logger.Errorf("failed to close ICE agent: %v", err)
		return err
	}
	return nil
}

func (agent *Agent) GetLocalUserCredentials() (string, string, error) {
	if agent.iceAgent == nil {
		return "", "", errors.New("ICE agent is not initialized")
	}
	return agent.iceAgent.GetLocalUserCredentials()
}

func (agent *Agent) GetRemoteCandidates() ([]ice.Candidate, error) {
	if agent.iceAgent == nil {
		return nil, errors.New("ICE agent is not initialized")
	}
	return agent.iceAgent.GetRemoteCandidates()
}

// AgentConfig holds the configuration for creating a new ICE agent.
type AgentConfig struct {
	StunUrl         string
	UniversalUdpMux *ice.UniversalUDPMuxDefault
}
