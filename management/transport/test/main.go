package main

import (
	"fmt"
	"os"
	"strconv"
	"wireflow/internal/infra"
	nats2 "wireflow/management/nats"
	"wireflow/management/transport"

	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	args := os.Args
	localId := args[1]
	remoteId := args[2]
	p := args[3]
	port, err := strconv.Atoi(p)
	if err != nil {
		panic(err)
	}
	ctx := signals.SetupSignalHandler()
	nats, err := nats2.NewNatsService(ctx, "nats://81.68.109.143:4222")
	if err != nil {
		panic(err)
	}

	peerManager := infra.NewPeerManager()
	conn, _, err := infra.ListenUDP("udp", uint16(port))
	dialer := transport.NewIceDialer(&transport.ICEDialerConfig{
		Sender:                 nats.Send,
		LocalId:                localId,
		RemoteId:               remoteId,
		UniversalUdpMuxDefault: infra.NewUdpMux(conn, false),
		PeerManager:            peerManager,
	})

	peerManager.AddPeer(localId, &infra.Peer{
		PublicKey: localId,
	})

	if err = nats.Subscribe(fmt.Sprintf("%s.%s", "wireflow.signals.peers", localId), dialer.HandleSignal); err != nil {
		panic(err)
	}

	if err = dialer.Prepare(ctx, remoteId); err != nil {
		panic(err)
	}

	connection, err := dialer.Dial(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("Dial successfully", connection.RemoteAddr())
}
