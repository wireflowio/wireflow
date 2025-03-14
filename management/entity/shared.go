package entity

import (
	"gorm.io/gorm"
	"time"
)

type GroupShared struct {
	gorm.Model
	UserId      uint
	GroupId     uint
	OwnerID     uint `gorm:"column:owner_id;size:20" json:"ownerId"`
	Description string
	GrantedAt   time.Time
	RevokedAt   time.Time
}

func (GroupShared) TableName() string {
	return "la_group_shared"
}

type PolicyShared struct {
	gorm.Model
	UserId      uint
	PolicyId    uint
	OwnerId     uint
	Description string
	GrantedAt   time.Time
	RevokedAt   time.Time
}

func (PolicyShared) TableName() string {
	return "la_policy_shared"
}

type NodeShared struct {
	gorm.Model
	UserId      uint
	NodeId      uint
	OwnerId     uint
	Description string
	GrantedAt   time.Time
	RevokedAt   time.Time
}

func (NodeShared) TableName() string {
	return "la_node_shared"
}

type LabelShared struct {
	gorm.Model
	UserId      uint
	LabelId     uint
	OwnerId     uint
	Description string
	GrantedAt   time.Time
	RevokedAt   time.Time
}

func (LabelShared) TableName() string {
	return "la_label_shared"
}
