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
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("oauth")

type AuthCodeRequest struct {
	AccessType   string
	Scope        string
	RedirectUri  string
	CodeVerifier string
	ClientId     string
}

func (req AuthCodeRequest) GetLoginUrl(ctx context.Context, baseLoginUrl string) (string, error) {
	ctx, span := tracer.Start(ctx, "AuthCodeRequest:getLoginUrl")
	defer span.End()

	endpoint, err := url.Parse(baseLoginUrl)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse base login url")

		return "", err
	}

	values := endpoint.Query()
	values.Add("client_id", req.ClientId)
	values.Add("access_type", req.AccessType)
	values.Add("scope", req.Scope)
	values.Add("code_challenge", req.CodeVerifier)

	span.SetAttributes(
		attribute.KeyValue{
			Key:   "client_id",
			Value: attribute.StringValue(req.ClientId),
		},
		attribute.KeyValue{
			Key:   "access_type",
			Value: attribute.StringValue(req.AccessType),
		},
		attribute.KeyValue{
			Key:   "scope",
			Value: attribute.StringValue(req.Scope),
		},
		attribute.KeyValue{
			Key:   "code_challenge",
			Value: attribute.StringValue(req.CodeVerifier),
		},
	)

	nonce := make([]byte, 16)
	_, err = rand.Read(nonce)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to generate 16 random bytes")

		return "", err
	}

	state := hex.EncodeToString(nonce)
	values.Add("state", state)
	values.Add("response_type", "code")
	values.Add("prompt", "login")

	span.SetAttributes(
		attribute.KeyValue{
			Key:   "state",
			Value: attribute.StringValue(state),
		},
		attribute.KeyValue{
			Key:   "response_type",
			Value: attribute.StringValue("code"),
		},
		attribute.KeyValue{
			Key:   "prompt",
			Value: attribute.StringValue("login"),
		},
	)

	endpoint.RawQuery = values.Encode()

	return endpoint.String(), nil
}

type RefreshRequest struct {
	Scope        string
	ClientId     string
	RefreshToken string
}

func (req RefreshRequest) FormData(ctx context.Context, out io.Writer) {
	ctx, span := tracer.Start(ctx, "RefreshRequest:formData")
	defer span.End()

	writer := multipart.NewWriter(out)
	writer.WriteField("grant_type", "refresh_token")
	writer.WriteField("client_id", req.ClientId)
	writer.WriteField("scope", req.Scope)
	writer.WriteField("refresh_token", req.RefreshToken)

	span.SetAttributes(
		attribute.KeyValue{
			Key:   "grant_type",
			Value: attribute.StringValue("refresh_token"),
		},
		attribute.KeyValue{
			Key:   "client_id",
			Value: attribute.StringValue(req.ClientId),
		},
		attribute.KeyValue{
			Key:   "scope",
			Value: attribute.StringValue(req.Scope),
		},
		attribute.KeyValue{
			Key:   "refresh_token",
			Value: attribute.StringValue(req.RefreshToken),
		},
	)
}

var globalClient = resty.New()

type OpenIdToken struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	IdToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
}

func (token OpenIdToken) Refresh(ctx context.Context, baseRefreshUrl, clientId string) (string, OpenIdToken, error) {
	ctx, span := tracer.Start(ctx, "OpenIdToken:refresh")
	defer span.End()

	span.SetAttributes(
		attribute.KeyValue{
			Key:   "scope",
			Value: attribute.StringValue(token.Scope),
		},
		attribute.KeyValue{
			Key:   "clientId",
			Value: attribute.StringValue(clientId),
		},
	)

	req := RefreshRequest{
		Scope:        token.Scope,
		ClientId:     clientId,
		RefreshToken: token.RefreshToken,
	}

	body := bytes.NewBuffer(nil)
	req.FormData(ctx, body)

	res, err := globalClient.R().
		SetBody(body).
		Post(baseRefreshUrl)
	if err != nil {
		return "", OpenIdToken{}, err
	}

	resToken := res.Body()

	var tokenResponse OpenIdToken
	err = json.Unmarshal(resToken, &tokenResponse)
	return string(resToken), tokenResponse, err
}
