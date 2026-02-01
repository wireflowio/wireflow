package controller

import (
	"context"
	"wireflow/internal/log"
	"wireflow/management/dto"
	"wireflow/management/service"
)

type UserController interface {
	Register(ctx context.Context, userDto dto.UserDto) error
}

var (
	_ UserController = (*userController)(nil)
)

type userController struct {
	log         *log.Logger
	userService service.UserService
}

func NewUserController() UserController {
	return &userController{
		log:         log.GetLogger("user-controller"),
		userService: service.NewUserService(),
	}
}

func (u userController) Register(ctx context.Context, userDto dto.UserDto) error {
	return u.userService.Register(ctx, userDto)
}
