package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"

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
