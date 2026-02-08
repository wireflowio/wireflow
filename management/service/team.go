package service

import (
	"context"
	"fmt"
	"wireflow/internal/log"
	"wireflow/management/database"
	"wireflow/management/model"

	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	client_r "wireflow/management/resource"
)

type TeamService interface {
	OnboardExternalUser(ctx context.Context, userId, extEmail, namespace string) (*model.User, error)
}

type teamService struct {
	log       *log.Logger
	K8sClient client.Client
	DB        *gorm.DB // SQLite 实例
}

func NewTeamService(k8sClient *client_r.Client) TeamService {
	return &teamService{
		log:       log.GetLogger("team-service"),
		K8sClient: k8sClient,
		DB:        database.DB,
	}
}

// OnboardExternalUser 当外部用户通过 SSO 登录时触发
func (s *teamService) OnboardExternalUser(ctx context.Context, userId, extEmail, namespace string) (*model.User, error) {
	var user model.User
	// 1. 同步外部用户到 SQLite
	err := s.DB.FirstOrCreate(&user, model.User{Email: extEmail, Namespace: namespace}).Error
	if err != nil {
		return nil, err
	}

	// 2. 如果用户没有团队，创建一个默认团队 (Namespace)
	var membership model.TeamMember
	err = s.DB.Where("user_id = ?", user.ID).First(&membership).Error
	if err == gorm.ErrRecordNotFound {
		return &user, s.CreateTeamWithInfrastructure(ctx, user.ID, "Default Team")
	}
	return &user, err
}

// CreateTeamWithInfrastructure 创建 K8s Namespace、Quota、RoleBinding
func (s *teamService) CreateTeamWithInfrastructure(ctx context.Context, ownerID string, teamName string) error {
	teamID := fmt.Sprintf("wf-team-%s", ownerID[:8]) // 确保 ID 兼容 DNS 命名

	return s.DB.Transaction(func(tx *gorm.DB) error {
		// --- A. SQLite 事务 ---
		team := model.Team{DisplayName: teamName}

		if err := tx.Create(&team).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.TeamMember{TeamID: teamID, UserID: ownerID, Role: "admin"}).Error; err != nil {
			return err
		}

		// --- B. K8s 基础设施下发 ---

		// 1. 创建 Namespace
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name:   teamID,
			Labels: map[string]string{"managed-by": "wireflow"},
		}}
		if err := s.K8sClient.Create(ctx, ns); err != nil {
			return err
		}

		// 2. 创建 ResourceQuota (限制 CRD 数量)
		quota := &corev1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{Name: "team-quota", Namespace: teamID},
			Spec: corev1.ResourceQuotaSpec{
				Hard: corev1.ResourceList{
					corev1.ResourceName("count/nodes.wireflow.io"): resource.MustParse("50"),
					corev1.ResourceSecrets:                         resource.MustParse("20"),
				},
			},
		}
		if err := s.K8sClient.Create(ctx, quota); err != nil {
			return err
		}

		// 3. 创建 RoleBinding (将 Owner 绑定到 Namespace)
		rb := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "owner-binding", Namespace: teamID},
			Subjects: []rbacv1.Subject{{
				Kind:     "User",
				Name:     ownerID, // 这里对应外部身份系统的 ID/Email
				APIGroup: "rbac.authorization.k8s.io",
			}},
			RoleRef: rbacv1.RoleRef{
				Kind:     "ClusterRole",
				Name:     "wf-resource-editor", // 引用之前定义的模板
				APIGroup: "rbac.authorization.k8s.io",
			},
		}
		return s.K8sClient.Create(ctx, rb)
	})
}
