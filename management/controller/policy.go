package controller

import (
	"context"
	"wireflow/management/dto"
	"wireflow/management/vo"
)

type PolicyController interface {
	ListPolicy(ctx context.Context, pageParam *dto.PageRequest) (*dto.PageResult[vo.PolicyVo], error)
	UpdatePolicy(ctx context.Context, peerDto *dto.PeerDto) (*vo.PolicyVo, error)
}
