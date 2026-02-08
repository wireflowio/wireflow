package model

import (
	"github.com/golang-jwt/jwt/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Token struct {
	Token      string      `json:"token"`
	Namespace  string      `json:"namespace"`
	UsageLimit int         `json:"usageLimit"`
	Expiry     metav1.Time `json:"expiry"`
	BoundPeers []string    `json:"boundPeers,omitempty"`
}

// WireFlowClaims 通常在 Dex 回调成功后，签发一个属于 WireFlow 自己的轻量级 JWT。
type WireFlowClaims struct {
	UserID string `json:"uid"`
	Email  string `json:"email"`
	// 增加当前选中的团队 ID，方便实现“Vercel 风格”的上下文切换
	TeamID    string `json:"tid"`
	Namespace string `json:"ns"`
	jwt.RegisteredClaims
}
