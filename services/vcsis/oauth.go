package vcsis

import (
	"vcassist-backend/pkg/openid"
	keychainv1 "vcassist-backend/proto/vcassist/services/keychain/v1"
)

type OAuthConfig struct {
	BaseLoginUrl string
	RefreshUrl   string
	ClientId     string
}

func (o OAuthConfig) GetOAuthFlow() (*keychainv1.OAuthFlow, error) {
	codeVerifier, err := oauth.GenerateCodeVerifier()
	if err != nil {
		return nil, err
	}
	return &keychainv1.OAuthFlow{
		BaseLoginUrl:    o.BaseLoginUrl,
		AccessType:      "offline",
		Scope:           "openid email profile",
		RedirectUri:     "com.powerschool.portal://",
		CodeVerifier:    codeVerifier,
		ClientId:        o.ClientId,
		TokenRequestUrl: "https://oauth2.googleapis.com/token",
	}, nil
}
