package service

import (
	"context"
	"wireflow/internal/log"
	"wireflow/management/database"
	"wireflow/management/dto"
	"wireflow/management/repository"
)

type UserService interface {
	Register(ctx context.Context, userDto dto.UserDto) error
}

type userService struct {
	log            *log.Logger
	userRepository repository.UserRepository
}

func (u userService) Register(ctx context.Context, userDto dto.UserDto) error {
	return u.userRepository.Register(ctx, &userDto)
}

func NewUserService() UserService {
	return &userService{
		log:            log.GetLogger("user-service"),
		userRepository: repository.NewUserRepository(database.DB),
	}
}
