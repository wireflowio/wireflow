package controller

import (
	"context"
	"wireflow/management/model"
	"wireflow/management/resource"
	"wireflow/management/service"
)

type TeamController interface {
	OnboardExternalUser(ctx context.Context, extEmail string) (*model.User, error)
}

type teamController struct {
	teamService service.TeamService
}

func (t teamController) OnboardExternalUser(ctx context.Context, extEmail string) (*model.User, error) {
	//TODO implement me
	panic("implement me")
}

func NewTeamController(client *resource.Client) TeamController {
	return &teamController{
		teamService: service.NewTeamService(client),
	}
}
