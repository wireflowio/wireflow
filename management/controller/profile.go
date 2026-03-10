package controller

import (
	"context"
	"wireflow/management/dto"
	"wireflow/management/service"
)

type ProfileController interface {
	GetProfile(ctx context.Context, userID string) (*dto.UserSettingsResponse, error)
	UpdateProfile(ctx context.Context, userID string, req dto.UpdateSettingsRequest) error
}

type profileController struct {
	profileService service.ProfileService
}

func (p profileController) GetProfile(ctx context.Context, userID string) (*dto.UserSettingsResponse, error) {
	return p.profileService.GetProfile(ctx, userID)
}

func (p profileController) UpdateProfile(ctx context.Context, userID string, req dto.UpdateSettingsRequest) error {
	return p.profileService.UpdateProfile(ctx, userID, req)
}

func NewProfileController() ProfileController {
	return &profileController{profileService: service.NewProfileService()}
}
