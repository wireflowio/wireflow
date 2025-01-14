package internal

import "linkany/pkg/config"

type ControlClient interface {
	// Register will register a device to linkany center
	Register() (*config.DeviceConf, error)

	Login(user *config.User) (*config.User, error)

	FetchPeers() (*config.DeviceConf, error)

	GetUsers() []*config.User
}
