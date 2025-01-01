package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	openidv1 "vcassist-backend/api/openid/v1"

	"github.com/go-resty/resty/v2"
	"google.golang.org/protobuf/encoding/protojson"
)

type Endpoints struct {
	UserInfo     string
	RefreshToken string
}

type Client struct {
	endpoints Endpoints
	clientId  string
	http      *resty.Client
}

func NewClient(endpoints Endpoints, clientId string) Client {
	return Client{
		endpoints: endpoints,
		clientId:  clientId,
		http:      resty.New(),
	}
}

type googleUserInfo struct {
	Email string `json:"email"`
}

// GetEmail gets the email associated with a token (if this succeeds this implies the token is valid).
func (client Client) GetEmail(ctx context.Context, token *openidv1.Token) (email string, err error) {
	res, err := client.http.R().
		SetContext(ctx).
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", token.GetAccessToken())).
		Get(client.endpoints.UserInfo)
	if err != nil {
		return "", err
	}
	if res.StatusCode() >= 400 || res.StatusCode() < 500 {
		return "", fmt.Errorf("invalid token")
	}
	body := res.Body()

	var result googleUserInfo
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}

	return result.Email, nil
}

// RefreshToken refreshes a powerschool access token.
func (client Client) RefreshToken(ctx context.Context, token *openidv1.Token) (*openidv1.Token, error) {
	form := url.Values{}
	form.Add("grant_type", "refresh_token")
	form.Add("client_id", client.clientId)
	form.Add("scope", token.GetScope())
	form.Add("refresh_token", token.GetRefreshToken())

	res, err := client.http.R().
		SetContext(ctx).
		SetBody(form.Encode()).
		SetHeader("content-type", "application/x-www-form-urlencoded").
		Post(client.endpoints.RefreshToken)
	if err != nil {
		return nil, err
	}

	var refreshed openidv1.Token
	err = protojson.Unmarshal(res.Body(), &refreshed)
	if err != nil {
		return nil, err
	}
	refreshed.RefreshToken = token.GetRefreshToken()

	return &refreshed, nil
}
