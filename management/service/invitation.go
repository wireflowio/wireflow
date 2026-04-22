package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"
	"wireflow/internal/store"
	"wireflow/management/dto"
	"wireflow/management/models"

	"gorm.io/gorm"
)

// InvitationService manages workspace invitations.
type InvitationService interface {
	Create(ctx context.Context, workspaceID, inviterID, email string, role dto.WorkspaceRole) (*models.WorkspaceInvitation, error)
	Accept(ctx context.Context, token, acceptorUserID string) error
	Revoke(ctx context.Context, invitationID string) error
	List(ctx context.Context, workspaceID string) ([]*models.WorkspaceInvitation, error)
}

type invitationService struct {
	store store.Store
}

func NewInvitationService(st store.Store) InvitationService {
	return &invitationService{store: st}
}

func (s *invitationService) Create(ctx context.Context, workspaceID, inviterID, email string, role dto.WorkspaceRole) (*models.WorkspaceInvitation, error) {
	// Verify inviter is a workspace admin.
	inviterMember, err := s.store.WorkspaceMembers().GetMembership(ctx, workspaceID, inviterID)
	if err != nil {
		return nil, errors.New("inviter is not a member of this workspace")
	}
	if dto.GetRoleWeight(inviterMember.Role) < dto.GetRoleWeight(dto.RoleAdmin) {
		return nil, errors.New("only admins can invite members")
	}
	if dto.GetRoleWeight(role) > dto.GetRoleWeight(inviterMember.Role) {
		return nil, errors.New("cannot invite with a role higher than your own")
	}

	// Prevent duplicate pending invitations.
	existing, err := s.store.WorkspaceInvitations().GetPendingByEmailAndWorkspace(ctx, email, workspaceID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("a pending invitation already exists for this email")
	}

	token, err := generateInviteToken()
	if err != nil {
		return nil, err
	}

	inv := &models.WorkspaceInvitation{
		WorkspaceID: workspaceID,
		InviterID:   inviterID,
		Email:       email,
		Role:        role,
		Token:       token,
		Status:      "pending",
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
	}
	if err := s.store.WorkspaceInvitations().Create(ctx, inv); err != nil {
		return nil, err
	}
	return inv, nil
}

func (s *invitationService) Accept(ctx context.Context, token, acceptorUserID string) error {
	return s.store.Tx(ctx, func(st store.Store) error {
		inv, err := st.WorkspaceInvitations().GetByToken(ctx, token)
		if err != nil {
			return errors.New("invitation not found")
		}
		if inv.Status != "pending" {
			return errors.New("invitation is no longer pending")
		}
		if time.Now().After(inv.ExpiresAt) {
			_ = st.WorkspaceInvitations().UpdateStatus(ctx, inv.ID, "expired")
			return errors.New("invitation has expired")
		}

		now := time.Now()
		if err := st.WorkspaceMembers().AddMember(ctx, &models.WorkspaceMember{
			WorkspaceID: inv.WorkspaceID,
			UserID:      acceptorUserID,
			Role:        inv.Role,
			Status:      "active",
			InvitedBy:   inv.InviterID,
			JoinedAt:    &now,
		}); err != nil {
			return err
		}
		return st.WorkspaceInvitations().UpdateStatus(ctx, inv.ID, "accepted")
	})
}

func (s *invitationService) Revoke(ctx context.Context, invitationID string) error {
	return s.store.WorkspaceInvitations().UpdateStatus(ctx, invitationID, "revoked")
}

func (s *invitationService) List(ctx context.Context, workspaceID string) ([]*models.WorkspaceInvitation, error) {
	return s.store.WorkspaceInvitations().ListByWorkspace(ctx, workspaceID)
}

func generateInviteToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
