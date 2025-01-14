package control

import "linkany/pkg/config"

type Api interface {
	//Login login will return token
	Login(user config.User) (config.User, error)

	//Logout logout will logout user
	Logout(user config.User) error

	//Register register will register device to linkany center
	Register()
}
