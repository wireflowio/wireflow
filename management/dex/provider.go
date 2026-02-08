package dex

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

var (
	verifier     *oidc.IDTokenVerifier
	oauth2Config oauth2.Config
)

func InitDex(ctx context.Context) {
	// 1. 发现 Dex 服务（从配置中读取 issuer URL）
	provider, _ := oidc.NewProvider(ctx, "http://dex.wireflow.local:5556/dex")

	// 2. 初始化 Verifier 用于后续校验 Token
	verifier = provider.Verifier(&oidc.Config{ClientID: "wireflow-bff"})

	// 3. 配置 OAuth2
	oauth2Config = oauth2.Config{
		ClientID:     "wireflow-bff",
		ClientSecret: "<BFF_CLIENT_SECRET>",
		Endpoint:     provider.Endpoint(),
		RedirectURL:  "http://wireflow.io/api/v1/auth/callback",
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
}
