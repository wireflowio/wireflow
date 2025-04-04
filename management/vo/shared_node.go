package vo

import (
	"time"

	"gorm.io/gorm"
)

type SharedNodeGroupVo struct {
	*GroupRelationVo
	ID          uint           `json:"id"`
	GroupId     uint           `json:"groupId"`
	Name        string         `json:"name"`
	NodeCount   int            `json:"nodeCount"`
	Status      string         `json:"status"`
	Description string         `json:"description"`
	CreatedAt   time.Time      `json:"createdAt"`
	DeletedAt   gorm.DeletedAt `json:"deletedAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	CreatedBy   string         `json:"createdBy"`
	UpdatedBy   string         `json:"updatedBy"`
}
