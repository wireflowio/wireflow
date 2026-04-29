package controller

import (
	"context"

	"github.com/alatticeio/lattice/internal/store"
	"github.com/alatticeio/lattice/management/resource"
	"github.com/alatticeio/lattice/management/service"
)

type TokenController interface {
	Create(ctx context.Context) (string, error)
	Delete(ctx context.Context, token string) error
}

type tokenController struct {
	tokenService service.TokenService
}

func (t *tokenController) Delete(ctx context.Context, token string) error {
	return t.tokenService.Delete(ctx, token)
}

func (t *tokenController) Create(ctx context.Context) (string, error) {
	return t.tokenService.Create(ctx)
}

func NewTokenController(client *resource.Client, st store.Store) TokenController {
	return &tokenController{
		tokenService: service.NewTokenService(client, st),
	}
}
