package oauth

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/url"

	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type AuthCodeRequest struct {
	AccessType   string
	Scope        string
	RedirectUri  string
	CodeVerifier string
	ClientId     string
}

func (req AuthCodeRequest) GetLoginUrl(baseLoginUrl string) (string, error) {
	endpoint, err := url.Parse(baseLoginUrl)
	if err != nil {
		return "", err
	}

	values := endpoint.Query()
	values.Add("client_id", req.ClientId)
	values.Add("access_type", req.AccessType)
	values.Add("scope", req.Scope)
	values.Add("code_challenge", req.CodeVerifier)

	nonce := make([]byte, 16)
	_, err = rand.Read(nonce)
	if err != nil {
		return "", err
	}
	values.Add("state", hex.EncodeToString(nonce))
	values.Add("response_type", "code")
	values.Add("prompt", "login")

	endpoint.RawQuery = values.Encode()

	return endpoint.String(), nil
}

type RefreshRequest struct {
	Scope        string
	ClientId     string
	RefreshToken string
}

func (req RefreshRequest) FormData(out io.Writer) {
	writer := multipart.NewWriter(out)
	writer.WriteField("grant_type", "refresh_token")
	writer.WriteField("client_id", req.ClientId)
	writer.WriteField("scope", req.Scope)
	writer.WriteField("refresh_token", req.RefreshToken)
}

var tracer = otel.Tracer("oauth")

var globalClient = resty.New()

type OpenIdToken struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	IdToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
}

func (token OpenIdToken) Refresh(ctx context.Context, baseRefreshUrl, clientId string) (OpenIdToken, error) {
	ctx, span := tracer.Start(ctx, "openidtoken:refresh")
	defer span.End()

	span.SetAttributes(
		attribute.KeyValue{
			Key:   "custom.scope",
			Value: attribute.StringValue(token.Scope),
		},
		attribute.KeyValue{
			Key:   "custom.clientId",
			Value: attribute.StringValue(clientId),
		},
	)

	req := RefreshRequest{
		Scope:        token.Scope,
		ClientId:     clientId,
		RefreshToken: token.RefreshToken,
	}

	body := bytes.NewBuffer(nil)
	req.FormData(body)

	res, err := globalClient.R().
		SetBody(body).
		Post(baseRefreshUrl)
	if err != nil {
		return OpenIdToken{}, err
	}

	var tokenResponse OpenIdToken
	err = json.Unmarshal(res.Body(), &tokenResponse)
	return tokenResponse, err
}
