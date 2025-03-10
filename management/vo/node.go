package vo

import (
	"gorm.io/gorm"
	"time"
)

type NodeGroupVo struct {
	ID          uint           `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	CreatedBy   string         `json:"created_by"`
	UpdatedBy   string         `json:"updated_by"`
}

type NodeVo struct {
	ID                  uint   `json:"id,string"`
	Name                string `json:"name,omitempty"`
	Description         string `json:"description,omitempty"`
	GroupID             uint   `json:"groupID,omitempty"`   // belong to which group
	CreatedBy           uint   `json:"createdBy,omitempty"` // ownerID
	UserID              int64  `json:"user_id,omitempty"`
	Hostname            string `json:"hostname,omitempty"`
	AppID               string `json:"app_id,omitempty"`
	Address             string `json:"address,omitempty"`
	Endpoint            string `json:"endpoint,omitempty"`
	PersistentKeepalive int    `json:"persistent_keepalive,omitempty"`
	PublicKey           string `json:"public_key,omitempty"`
	AllowedIPs          string `json:"allowed_ips,omitempty"`
	RelayIP             string `json:"relay_ip,omitempty"`
	TieBreaker          int64  `json:"tie_breaker"`
	Ufrag               string `json:"ufrag"`
	Pwd                 string `json:"pwd"`
	Port                int    `json:"port"`
	Status              int    `json:"status"`
}
