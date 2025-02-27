package entity

import (
	"gorm.io/gorm"
	"time"
)

// invtes invite others
type Invites struct {
	gorm.Model
	InvitationId int64 // invitation user id
	InviterId    int64 // inviter user id
	MobilePhone  string
	Email        string
	AcceptStatus AcceptStatus
	InvitedAt    time.Time
	CanceledAt   time.Time
}

// Invitation user invite other join its network
type Invitation struct {
	gorm.Model
	InvitationId int64 // invitation user id
	InviterId    int64 // inviter user id
	MobilePhone  string
	Email        string
	AcceptStatus AcceptStatus
	Network      string //192.168.0.0/24
	InvitedAt    time.Time
	AcceptAt     time.Time
}

func (i *Invites) TableName() string {
	return "la_user_invites"
}

func (i *Invitation) TableName() string {
	return "la_user_invitations"
}

type AcceptStatus int

const (
	NewInvite = iota
	Accept
	Rejected
	Canceled
)

func (a AcceptStatus) String() string {
	switch a {
	case NewInvite:
		return "待处理"
	case Accept:
		return "已接受"
	case Rejected:
		return "已拒绝"
	case Canceled:
		return "已取消"
	default:
		return "unknown"
	}
}
