package server

import (
	"linkany/management/grpc/mgt"
	"linkany/management/utils"
)

func CreateChannel(pubKey string) chan *mgt.WatchMessage {
	manager := utils.NewWatchManager()
	ch := make(chan *mgt.WatchMessage)
	manager.Add(pubKey, make(chan *mgt.WatchMessage))

	return ch
}
