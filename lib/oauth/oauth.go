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

var tracer = otel.Tracer("vcassist.lib.oauth")

type AuthCodeRequest struct {
	AccessType   string
	Scope        string
	RedirectUri  string
	CodeVerifier string
	ClientId     string
}

func GetLoginUrl(ctx context.Context, req AuthCodeRequest, baseLoginUrl string) (string, error) {
	ctx, span := tracer.Start(ctx, "GetLoginUrl")
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
	values.Add("redirect_uri", req.RedirectUri)

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
		attribute.KeyValue{
			Key:   "redirect_uri",
			Value: attribute.StringValue(req.RedirectUri),
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
	ctx, span := tracer.Start(ctx, "RefreshRequest:FormData")
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

func Refresh(ctx context.Context, token OpenIdToken, baseRefreshUrl, clientId string) (string, OpenIdToken, error) {
	ctx, span := tracer.Start(ctx, "OpenIdToken:Refresh")
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
		SetContext(ctx).
		SetBody(body).
		Post(baseRefreshUrl)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", OpenIdToken{}, err
	}

	resToken := res.Body()

	var tokenResponse OpenIdToken
	err = json.Unmarshal(resToken, &tokenResponse)
	return string(resToken), tokenResponse, err
}

type TokenRequest struct {
	ClientId     string `json:"client_id"`
	Scope        string `json:"scope"`
	AuthCode     string `json:"code"`
	CodeVerifier string `json:"code_verifier,omitempty"`
	RedirectUri  string `json:"redirect_uri"`
	GrantType    string `json:"grant_type"`
}

// note: the GrantType field only exists as part of the json request, it should not be set
func GetToken(ctx context.Context, req TokenRequest, tokenRequestUrl string) (string, OpenIdToken, error) {
	ctx, span := tracer.Start(ctx, "GetToken")
	defer span.End()

	req.GrantType = "authorization_code"
	body, err := json.Marshal(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to marshal token request")
		return "", OpenIdToken{}, err
	}

	res, err := globalClient.R().
		SetBody(body).
		Post(tokenRequestUrl)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch token")
		return "", OpenIdToken{}, err
	}

	var token OpenIdToken
	err = json.Unmarshal(res.Body(), &token)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to unmarshal token json")
		return "", OpenIdToken{}, err
	}

	return res.String(), token, nil
}

func GenerateCodeVerifier() (string, error) {
	nonce := make([]byte, 32)
	_, err := rand.Read(nonce)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(nonce), nil
}
