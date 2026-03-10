package service

import (
	"context"
	"wireflow/management/database"
	"wireflow/management/dto"
	"wireflow/management/models"
	"wireflow/management/repository"

	"gorm.io/gorm"
)

// ProfileService 定义个人信息业务接口
type ProfileService interface {
	// GetProfile 获取聚合后的用户信息
	GetProfile(ctx context.Context, userID string) (*dto.UserSettingsResponse, error)

	// UpdateProfile 更新用户信息（涉及多表事务）
	UpdateProfile(ctx context.Context, userID string, req dto.UpdateSettingsRequest) error
}

type profileService struct {
	// 这里注入 DB 实例或 Repository
	db          *gorm.DB
	profileRepo *repository.ProfileRepository
}

func NewProfileService() ProfileService {
	return &profileService{db: database.DB,
		profileRepo: repository.NewProfileRepository(database.DB)}
}

func (s *profileService) GetProfile(ctx context.Context, userID string) (*dto.UserSettingsResponse, error) {
	var user models.User
	var profile models.UserProfile

	// 使用事务或并行查询（这里演示标准查询）
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return nil, err
	}

	// 如果 profile 不存在则创建默认的
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).FirstOrCreate(&profile, models.UserProfile{UserID: userID}).Error; err != nil {
		return nil, err
	}

	// 转换为前端 Store 期待的结构
	return &dto.UserSettingsResponse{
		Name:        user.Username,
		Email:       user.Email,
		AvatarURL:   user.Avatar,
		Title:       profile.Title,
		Company:     profile.Company,
		Bio:         profile.Bio,
		Timezone:    profile.Timezone,
		Language:    profile.Language,
		EmailNotify: profile.EmailNotify,
	}, nil
}

func (s *profileService) UpdateProfile(ctx context.Context, userID string, req dto.UpdateSettingsRequest) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		// 1. 更新基础账号信息 (User 表)
		user := models.User{}
		user.ID = userID
		user.Email = req.Email
		user.Avatar = req.AvatarURL
		user.Address = req.Address
		user.Gender = req.Gender

		userRepo := repository.NewUserRepository(tx)
		err := userRepo.Update(ctx, &user)
		if err != nil {
			return err
		}

		// 2. 更新扩展配置 (UserProfile 表)
		// 使用 Updates 结构体或 Map，GORM 会自动忽略零值（取决于你的业务逻辑）
		profileUpdates := models.UserProfile{
			UserID:      userID,
			Title:       req.Title,
			Company:     req.Company,
			Bio:         req.Bio,
			Timezone:    req.Timezone,
			Language:    req.Language,
			EmailNotify: req.EmailNotify,
		}

		profileRepo := repository.NewProfileRepository(tx)
		profiles, err := profileRepo.Find(ctx, repository.WithUserID(profileUpdates.UserID))
		if err != nil {
			return err
		}

		if len(profiles) == 0 {
			err = profileRepo.Create(ctx, &profileUpdates)
			if err != nil {
				return err
			}
		}

		return profileRepo.Update(ctx, &profileUpdates)
	})
}
