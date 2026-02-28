package main

import (
	"wireflow/internal/infra"
	"wireflow/monitor"
)

func main() {
	peerManager := infra.NewPeerManager()
	runner := monitor.NewMonitorRunner(peerManager)
	runner.Run()
}
