package entity

import (
	"time"
)

type Peer struct {
	ID                  int64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	InstanceID          int64     `gorm:"column:instance_id" json:"instance_id"`
	UserID              int64     `gorm:"column:user_id" json:"user_id"`
	Name                string    `gorm:"column:name;size:20" json:"name"`
	Hostname            string    `gorm:"column:hostname;size:50" json:"hostname"`
	AppID               string    `gorm:"column:app_id;size:20" json:"app_id"`
	InsPrivateKey       string    `gorm:"column:ins_private_key;size:50" json:"ins_private_key"`
	InsPublicKey        string    `gorm:"column:ins_public_key;size:50" json:"ins_public_key"`
	Address             string    `gorm:"column:address;size:50" json:"address"`
	Endpoint            string    `gorm:"column:endpoint;size:50" json:"endpoint"`
	PersistentKeepalive int       `gorm:"column:persistent_keepalive" json:"persistent_keepalive"`
	PublicKey           string    `gorm:"column:public_key;size:50" json:"public_key"`
	PrivateKey          string    `gorm:"column:private_key;size:50" json:"private_key"`
	AllowedIPs          string    `gorm:"column:allowed_ips;size:50" json:"allowed_ips"`
	CreateDate          time.Time `gorm:"column:create_date" json:"create_date"`
	HostIP              string    `gorm:"column:host_ip;size:100" json:"host_ip"`
	SrflxIP             string    `gorm:"column:srflx_ip;size:100" json:"srflx_ip"`
	RelayIP             string    `gorm:"column:relay_ip;size:100" json:"relay_ip"`
	TieBreaker          uint64    `gorm:"column:tie_breaker" json:"tie_breaker"`
	UpdatedAt           time.Time `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt           time.Time `gorm:"column:deleted_at" json:"deleted_at"`
	CreatedAt           time.Time `gorm:"column:created_at" json:"created_at"`
	Ufrag               string    `gorm:"column:ufrag;size:30" json:"ufrag"`
	Pwd                 string    `gorm:"column:pwd;size:50" json:"pwd"`
	Port                int       `gorm:"column:port" json:"port"`
	Status              string    `gorm:"column:status" json:"status"`
	Online              int       `gorm:"column:online" json:"online"`
}

// TableName returns the table name of the model
func (Peer) TableName() string {
	return "la_peers"
}
