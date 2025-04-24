package repository

import (
	"context"
	"linkany/management/entity"

	"gorm.io/gorm"
)

type PermitRepository interface {
	WithTx(tx *gorm.DB) PermitRepository
	Create(ctx context.Context, accessPolicy *entity.AccessPolicy) error
	Delete(ctx context.Context, accessId uint64) error
	Find(ctx context.Context, accessId uint64) (*entity.AccessPolicy, error)
}

var (
	_ PermitRepository = (*accessRepository)(nil)
)

type accessRepository struct {
	db *gorm.DB
}

func NewAccessRepository(db *gorm.DB) PermitRepository {
	return &accessRepository{
		db: db,
	}
}

func (r *accessRepository) WithTx(tx *gorm.DB) PermitRepository {
	return &accessRepository{
		db: tx,
	}
}

func (r *accessRepository) Create(ctx context.Context, access *entity.AccessPolicy) error {
	return r.db.WithContext(ctx).Create(access).Error
}

func (r *accessRepository) Delete(ctx context.Context, accessId uint64) error {
	return r.db.WithContext(ctx).Delete(&entity.Node{}, accessId).Error
}

func (r *accessRepository) Find(ctx context.Context, accessId uint64) (*entity.AccessPolicy, error) {
	var access entity.AccessPolicy
	err := r.db.WithContext(ctx).First(&access, accessId).Error
	if err != nil {
		return nil, err
	}
	return &access, nil
}
