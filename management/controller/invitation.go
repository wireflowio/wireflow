package controller

import (
	"context"
	"wireflow/internal/store"
	"wireflow/management/dto"
	"wireflow/management/models"
	"wireflow/management/service"
)

type InvitationController interface {
	Create(ctx context.Context, workspaceID, inviterID, email string, role dto.WorkspaceRole) (*models.WorkspaceInvitation, error)
	Accept(ctx context.Context, token, userID string) error
	Revoke(ctx context.Context, invitationID string) error
	List(ctx context.Context, workspaceID string) ([]*models.WorkspaceInvitation, error)
}

type invitationController struct {
	svc service.InvitationService
}

func NewInvitationController(st store.Store) InvitationController {
	return &invitationController{svc: service.NewInvitationService(st)}
}

func (c *invitationController) Create(ctx context.Context, workspaceID, inviterID, email string, role dto.WorkspaceRole) (*models.WorkspaceInvitation, error) {
	return c.svc.Create(ctx, workspaceID, inviterID, email, role)
}

func (c *invitationController) Accept(ctx context.Context, token, userID string) error {
	return c.svc.Accept(ctx, token, userID)
}

func (c *invitationController) Revoke(ctx context.Context, invitationID string) error {
	return c.svc.Revoke(ctx, invitationID)
}

func (c *invitationController) List(ctx context.Context, workspaceID string) ([]*models.WorkspaceInvitation, error) {
	return c.svc.List(ctx, workspaceID)
}
