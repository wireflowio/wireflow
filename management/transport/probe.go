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

package transport

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"wireflow/internal/grpc"
	"wireflow/internal/infra"
	"wireflow/internal/log"

	"github.com/wireflowio/ice"
)

var (
	_ infra.Probe = (*Probe)(nil)
)

// Probe for probe connection from two peers.
type Probe struct {
	localId         string
	peerId          string
	factory         *TransportFactory
	ice             infra.Transport
	state           ice.ConnectionState
	signal          infra.SignalService
	ctx             context.Context
	cancel          context.CancelFunc
	closeAckOnce    sync.Once
	closeOnce       sync.Once
	answerChan      chan struct{}
	remoteOfferChan chan struct{}
	log             *log.Logger
	lastSeen        time.Time
	rtt             time.Duration
	handShackAck    chan struct{}

	// Add wrrp
	wrrp      infra.Transport
	wrrpReady atomic.Bool
	iceReady  atomic.Bool
	iceChan   chan struct{}
	wrrpChan  chan struct{}
}

func (p *Probe) OnConnectionStateChange(state ice.ConnectionState) {
	p.updateState(state)
	p.log.Info("Setting new connection status", "state", state)
}

func (p *Probe) Probe(ctx context.Context, remoteId string) error {
	if p.state == ice.ConnectionStateUnknown {
		return fmt.Errorf("invalid state.")
	}

	if p.state != ice.ConnectionStateNew {
		return nil
	}

	p.updateState(ice.ConnectionStateChecking)
	// 1. first prepare candidate then send to remoteId
	go func() {
		if err := p.Prepare(ctx, remoteId, p.signal.Send); err != nil {
			p.OnTransportFail(err)
		}
	}()

	return nil
}

// Prepare candidate, because using natsk, so we send offer directly when candidate ready.
func (p *Probe) Prepare(ctx context.Context, remoteId string, send func(ctx context.Context, remoteId string, data []byte) error) error {
	p.log.Info("Prepare probe peer", "remoteId", remoteId)
	defer p.updateState(ice.ConnectionStateChecking)
	go func() {
		err := p.Start(ctx, remoteId)
		if err != nil {
			p.log.Error("Start probe peer failed", err)
		}
	}()

	return p.ice.Prepare()
}

func (p *Probe) probeWrrpPacket(ctx context.Context, remoteId string, packetType grpc.PacketType) error {
	//packet := &grpc.SignalPacket{
	//	SenderId: p.localId,
	//	Type:     packetType,
	//	Payload: &grpc.SignalPacket_Handshake{
	//		Handshake: &grpc.Handshake{
	//			Timestamp: time.Now().Unix(),
	//		},
	//	},
	//}

	//data, err := proto.Marshal(packet)
	//if err != nil {
	//	return err
	//}
	//
	//sessionId, err := infra.IDFromPublicKey(remoteId)
	//if err != nil {
	//	return err
	//}
	return nil
}

func (p *Probe) HandleOffer(ctx context.Context, remoteId string, packet *grpc.SignalPacket) error {
	defer func() {
		p.closeOnce.Do(func() {
			close(p.remoteOfferChan)
		})
	}()

	return p.ice.HandleOffer(ctx, remoteId, packet)
}

func (p *Probe) Start(ctx context.Context, remoteId string) error {
	p.log.Info("Start probe peer", "remoteId", remoteId)
	sendReady, recvReady := false, false
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	go func() {
		for {
			select {
			case <-ctx.Done():
				p.log.Error("stop send ready ack", ctx.Err())
				return
			case <-p.remoteOfferChan:
				recvReady = true
			}

			//
			if sendReady && recvReady {
				p.log.Info("send ready and recv ready, will dial or accept connection")
				break
			}
		}

		if err := p.ice.Start(ctx, remoteId); err != nil {
			p.OnTransportFail(err)
		}
	}()

	return nil
}

func (p *Probe) Ping(ctx context.Context) error {
	return nil
}

func (p *Probe) OnSuccess(addr string) error {
	p.log.Info("OnSuccess", "addr", addr)
	return nil
}

func (p *Probe) OnTransportFail(err error) {
	p.log.Error("OnTransportFail", err)
}

func (p *Probe) updateState(state ice.ConnectionState) {
	p.state = state
}
