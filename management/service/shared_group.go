package service

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"linkany/management/dto"
	"linkany/management/entity"
	"linkany/management/utils"
	"linkany/management/vo"
	"linkany/pkg/log"
)

type SharedGroupService interface {
	//Group
	GetNodeGroup(ctx context.Context, id string) (*vo.NodeGroupVo, error)
	UpdateGroup(ctx context.Context, dto *dto.NodeGroupDto) error
	DeleteGroup(ctx context.Context, id string) error
	ListGroups(ctx context.Context, params *dto.GroupParams) (*vo.PageVo, error)
	QueryGroups(ctx context.Context, params *dto.GroupParams) ([]*vo.NodeGroupVo, error)

	ListGroupPolicy(ctx context.Context, params *dto.GroupPolicyParams) ([]*vo.GroupPolicyVo, error)
	DeleteGroupPolicy(ctx context.Context, groupId uint, policyId uint) error
	DeleteGroupNode(ctx context.Context, groupId uint, nodeId uint) error
}

var (
	_ SharedGroupService = (*sharedGroupServiceImpl)(nil)
)

type sharedGroupServiceImpl struct {
	logger *log.Logger
	*DatabaseService
	manager           *vo.WatchManager
	nodeServiceImpl   NodeService
	policyServiceImpl AccessPolicyService
}

// NodeGroup
func (g *sharedGroupServiceImpl) GetNodeGroup(ctx context.Context, nodeId string) (*vo.NodeGroupVo, error) {
	var (
		group entity.NodeGroup
		err   error
	)

	if err = g.Model(&entity.NodeGroup{}).Where("id = ?", nodeId).First(&group).Error; err != nil {
		return nil, err
	}

	res, err := g.fetchNodeAndGroup(group.ID)

	return &vo.NodeGroupVo{
		ID:          group.ID,
		Name:        group.Name,
		Description: group.Description,
		//NodeCount:   len(groupNodes),
		GroupRelationVo: res,
		CreatedAt:       group.CreatedAt,
		DeletedAt:       group.DeletedAt,
		UpdatedAt:       group.UpdatedAt,
		CreatedBy:       group.CreatedBy,
		UpdatedBy:       group.UpdatedBy,
	}, nil
}

func (g *sharedGroupServiceImpl) UpdateGroup(ctx context.Context, dto *dto.NodeGroupDto) error {
	return g.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		var group *entity.NodeGroup
		if group, err = updateSharedGroupData(ctx, tx, dto); err != nil {
			return err
		}

		return g.handleGP(ctx, tx, dto, group)
	})

}

func updateSharedGroupData(ctx context.Context, tx *gorm.DB, dto *dto.NodeGroupDto) (*entity.NodeGroup, error) {
	group := &entity.NodeGroup{
		Description: dto.Description,
		IsPublic:    dto.IsPublic,
		UpdatedBy:   dto.UpdatedBy,
	}

	if err := tx.Model(&entity.NodeGroup{}).Where("id = ?", dto.ID).Updates(group).Error; err != nil {
		return nil, err
	}

	// should add to
	group.ID = dto.ID

	return group, nil
}

func (g *sharedGroupServiceImpl) handleGP(ctx context.Context, tx *gorm.DB, dto *dto.NodeGroupDto, group *entity.NodeGroup) error {

	var err error
	if dto.NodeIdList != nil {
		for _, nodeId := range dto.NodeIdList {
			var groupNode entity.GroupNode
			if err = tx.Model(&entity.GroupNode{}).Where("group_id = ? and node_id = ?", group.ID, nodeId).First(&groupNode).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					var node entity.Node
					if err = tx.Model(&entity.Node{}).Where("id = ?", nodeId).Find(&node).Error; err != nil {
						return err
					}

					groupNode = entity.GroupNode{
						GroupID:   group.ID,
						NodeID:    node.ID,
						GroupName: group.Name,
						NodeName:  node.Name,
						CreatedBy: ctx.Value("username").(string),
					}
					if err := tx.Model(&entity.GroupNode{}).Create(&groupNode).Error; err != nil {
						return err
					}

					// add push message
					g.manager.Push(node.PublicKey, &vo.Message{
						EventType: vo.EventTypeGroupAdd,
						GroupMessage: &vo.GroupMessage{
							GroupName: group.Name,
							GroupId:   group.ID,
						},
					})
				}
			}
		}
	}

	if dto.PolicyIdList != nil {
		for _, policyId := range dto.PolicyIdList {
			var groupPolicy entity.GroupPolicy
			if err = tx.Model(&entity.GroupPolicy{}).Where("group_id = ? and policy_id = ?", group.ID, policyId).First(&groupPolicy).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					var policy entity.AccessPolicy
					if err = tx.Model(&entity.AccessPolicy{}).Where("id = ?", policyId).Find(&policy).Error; err != nil {
						return err
					}

					groupPolicy = entity.GroupPolicy{
						GroupID:    group.ID,
						PolicyId:   policy.ID,
						PolicyName: policy.Name,
						CreatedBy:  ctx.Value("username").(string),
					}
					if err := tx.Model(&entity.GroupPolicy{}).Create(&groupPolicy).Error; err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (g *sharedGroupServiceImpl) DeleteGroup(ctx context.Context, id string) error {
	return g.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		if err = tx.Model(&entity.NodeGroup{}).Where("id = ?", id).Delete(&entity.NodeGroup{}).Error; err != nil {
			return err
		}

		if err = tx.Model(&entity.GroupNode{}).Where("group_id = ?", id).Delete(&entity.GroupNode{}).Error; err != nil {
			return err
		}

		if err = tx.Model(&entity.GroupPolicy{}).Where("group_id = ?", id).Delete(&entity.GroupPolicy{}).Error; err != nil {
			return err
		}

		return nil
	})

}

func (g *sharedGroupServiceImpl) ListGroups(ctx context.Context, params *dto.GroupParams) (*vo.PageVo, error) {
	var nodeGroups []entity.NodeGroup

	result := new(vo.PageVo)
	sql, wrappers := utils.Generate(params)
	db := g.DB
	if sql != "" {
		db = db.Where(sql, wrappers)
	}

	if err := db.Model(&entity.NodeGroup{}).Count(&result.Total).Error; err != nil {
		return nil, err
	}

	g.logger.Verbosef("sql: %s, wrappers: %v", sql, wrappers)
	if err := db.Model(&entity.NodeGroup{}).Offset((params.Page - 1) * params.Size).Limit(params.Size).Find(&nodeGroups).Error; err != nil {
		return nil, err
	}

	var nodeVos []vo.NodeGroupVo
	for _, group := range nodeGroups {
		res, err := g.fetchNodeAndGroup(group.ID)
		if err != nil {
			return nil, err
		}
		nodeVos = append(nodeVos, vo.NodeGroupVo{
			ID:              group.ID,
			Name:            group.Name,
			Description:     group.Description,
			GroupRelationVo: res,
			CreatedAt:       group.CreatedAt,
			DeletedAt:       group.DeletedAt,
			UpdatedAt:       group.UpdatedAt,
			CreatedBy:       group.CreatedBy,
			UpdatedBy:       group.UpdatedBy,
		})
	}

	result.Data = nodeVos
	result.Current = params.Page
	result.Page = params.Page
	result.Size = params.Size

	return result, nil
}

func (g *sharedGroupServiceImpl) QueryGroups(ctx context.Context, params *dto.GroupParams) ([]*vo.NodeGroupVo, error) {
	var nodeGroups []entity.NodeGroup

	sql, wrappers := utils.Generate(params)
	db := g.DB
	if sql != "" {
		db = db.Where(sql, wrappers)
	}

	g.logger.Verbosef("sql: %s, wrappers: %v", sql, wrappers)
	if err := db.Model(&entity.NodeGroup{}).Offset((params.Page - 1) * params.Size).Limit(params.Size).Find(&nodeGroups).Error; err != nil {
		return nil, err
	}

	var nodeVos []*vo.NodeGroupVo
	for _, group := range nodeGroups {
		res, err := g.fetchNodeAndGroup(group.ID)
		if err != nil {
			return nil, err
		}
		nodeVos = append(nodeVos, &vo.NodeGroupVo{
			ID:              group.ID,
			Name:            group.Name,
			Description:     group.Description,
			GroupRelationVo: res,
			CreatedAt:       group.CreatedAt,
			DeletedAt:       group.DeletedAt,
			UpdatedAt:       group.UpdatedAt,
			CreatedBy:       group.CreatedBy,
			UpdatedBy:       group.UpdatedBy,
		})
	}

	return nodeVos, nil
}

func (g *sharedGroupServiceImpl) fetchNodeAndGroup(groupId uint) (*vo.GroupRelationVo, error) {
	// query group node
	var groupNodes []entity.GroupNode
	var err error
	if err = g.Model(&entity.GroupNode{}).Where("group_id = ?", groupId).Find(&groupNodes).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	result := new(vo.GroupRelationVo)
	nodeResourceVo := new(vo.NodeResourceVo)

	nodeValues := make(map[string]string, 1)
	for _, groupNode := range groupNodes {
		nodeValues[fmt.Sprintf("%d", groupNode.NodeID)] = groupNode.NodeName
	}
	nodeResourceVo.NodeValues = nodeValues
	result.NodeResourceVo = nodeResourceVo

	// query policies
	var groupPolicies []entity.GroupPolicy
	if err = g.Model(&entity.GroupPolicy{}).Where("group_id = ?", groupId).Find(&groupPolicies).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	policyResourceVo := new(vo.PolicyResourceVo)
	policyValues := make(map[string]string, 1)
	for _, groupPolicy := range groupPolicies {
		policyValues[fmt.Sprintf("%d", groupPolicy.PolicyId)] = groupPolicy.PolicyName
	}

	policyResourceVo.PolicyValues = policyValues
	result.PolicyResourceVo = policyResourceVo

	return result, nil
}

func (g *sharedGroupServiceImpl) ListGroupPolicy(ctx context.Context, params *dto.GroupPolicyParams) ([]*vo.GroupPolicyVo, error) {
	var groupPolicies []*entity.GroupPolicy
	if err := g.Model(&entity.GroupPolicy{}).Where("group_id = ?", params.GroupId).Find(&groupPolicies).Error; err != nil {
		return nil, err
	}

	var groupPolicyVos []*vo.GroupPolicyVo
	for _, groupPolicy := range groupPolicies {
		groupPolicyVos = append(groupPolicyVos, &vo.GroupPolicyVo{
			ID:          groupPolicy.ID,
			GroupId:     groupPolicy.GroupID,
			PolicyId:    groupPolicy.PolicyId,
			PolicyName:  groupPolicy.PolicyName,
			Description: groupPolicy.Description,
		})
	}
	return groupPolicyVos, nil
}

func (g *sharedGroupServiceImpl) DeleteGroupPolicy(ctx context.Context, groupId uint, policyId uint) error {
	return g.Model(&entity.GroupPolicy{}).Where("group_id = ? and policy_id = ?", groupId, policyId).Delete(&entity.GroupPolicy{}).Error
}

func (g *sharedGroupServiceImpl) DeleteGroupNode(ctx context.Context, groupId uint, nodeId uint) error {
	return g.DB.Transaction(func(tx *gorm.DB) error {
		var groupNode entity.GroupNode
		if err := g.Model(&entity.GroupNode{}).Where("group_id = ? and node_id = ?", groupId, nodeId).Delete(&groupNode).Error; err != nil {
			return err
		}

		var node entity.Node
		if err := g.Model(&entity.Node{}).Where("id = ?", nodeId).Find(&node).Error; err != nil {
			return err
		}

		g.manager.Push(node.PublicKey, &vo.Message{
			EventType: vo.EventTypeGroupRemove,
			GroupMessage: &vo.GroupMessage{
				GroupId:   groupId,
				GroupName: groupNode.GroupName,
			},
		})

		return nil
	})
}
