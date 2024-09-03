package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"vcassist-backend/lib/telemetry"
)

var tracer = telemetry.Tracer("vcassist.lib.oauth")

type AuthCodeRequest struct {
	AccessType   string
	Scope        string
	RedirectUri  string
	CodeVerifier string
	ClientId     string
}

func GetLoginUrl(ctx context.Context, req AuthCodeRequest, baseLoginUrl string) (string, error) {
	endpoint, err := url.Parse(baseLoginUrl)
	if err != nil {
		return "", err
	}

	values := endpoint.Query()
	values.Add("client_id", req.ClientId)
	values.Add("access_type", req.AccessType)
	values.Add("scope", req.Scope)
	values.Add("code_challenge", req.CodeVerifier)
	values.Add("redirect_uri", req.RedirectUri)

	nonce := make([]byte, 16)
	_, err = rand.Read(nonce)
	if err != nil {
		return "", err
	}

	state := hex.EncodeToString(nonce)
	values.Add("state", state)
	values.Add("response_type", "code")
	values.Add("prompt", "login")

	endpoint.RawQuery = values.Encode()

	return endpoint.String(), nil
}

type OpenIdToken struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	IdToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
}

type TokenRequest struct {
	ClientId     string `json:"client_id"`
	Scope        string `json:"scope"`
	AuthCode     string `json:"code"`
	CodeVerifier string `json:"code_verifier,omitempty"`
	RedirectUri  string `json:"redirect_uri"`
	GrantType    string `json:"grant_type"`
}

func GenerateCodeVerifier() (string, error) {
	nonce := make([]byte, 32)
	_, err := rand.Read(nonce)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(nonce), nil
}
