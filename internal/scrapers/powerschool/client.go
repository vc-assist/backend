package powerschool

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/cookiejar"
	"time"
	"vcassist-backend/internal/components/telemetry"

	"github.com/go-resty/resty/v2"
	"golang.org/x/time/rate"
)

const (
	report_client_login_oauth   = "client.login-oauth"
	report_client_graphql_query = "client.graphql-query"
)

type client struct {
	http *resty.Client
	tel  telemetry.API
}

func newClient(baseUrl string, tel telemetry.API) (*client, error) {
	tel = telemetry.NewScopedAPI("powerschool_scraper", tel)

	httpClient := resty.New()
	httpClient.SetTimeout(time.Minute)
	httpClient.SetBaseURL(baseUrl)
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	httpClient.SetCookieJar(jar)
	httpClient.SetHeader("user-agent", "okhttp/4.9.1")

	// 2 requests max per second
	// max burst >= 2 just means that no requests will be dropped
	rateLimiter := rate.NewLimiter(2, 2)
	httpClient.OnBeforeRequest(func(_ *resty.Client, req *resty.Request) error {
		err = rateLimiter.Wait(req.Context())
		if err != nil {
			return err
		}
		return nil
	})

	return &client{http: httpClient, tel: tel}, nil
}

type openIdToken struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	IdToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
}

func (c *client) LoginOAuth(ctx context.Context, token string) error {
	c.tel.ReportDebug(report_client_login_oauth, token)

	var openidToken openIdToken
	err := json.Unmarshal([]byte(token), &openidToken)
	if err != nil {
		c.tel.ReportBroken(
			report_client_login_oauth,
			fmt.Errorf("json unmarshal: %w", err),
		)
		return err
	}

	c.http.
		SetHeader("Authorization", fmt.Sprintf(
			"%s %s",
			openidToken.TokenType,
			openidToken.AccessToken,
		)).
		SetHeader("profileUri", openidToken.IdToken).
		SetHeader("ServerURL", c.http.BaseURL)

	return nil
}

func decodeBulletinTimestamp(tstr string) (time.Time, error) {
	return time.Parse("2024-08-13", tstr)
}

func decodeTimestamp(tstr string) (time.Time, error) {
	// aka. parse by ISO timestamp
	return time.Parse(time.RFC3339, tstr)
}

type graphqlRequest struct {
	Name     string `json:"operationName"`
	Query    string `json:"query"`
	Variable any    `json:"variables"`
}

type graphqlResponse[T any] struct {
	Data T `json:"data"`
}

func graphqlQuery[O any](
	ctx context.Context,
	client *client,
	name,
	query string,
	variables any,
	output *O,
) error {
	client.tel.ReportDebug(report_client_graphql_query, name, variables)

	body, err := json.Marshal(graphqlRequest{
		Name:     name,
		Query:    query,
		Variable: variables,
	})
	if err != nil {
		client.tel.ReportBroken(
			report_client_graphql_query,
			fmt.Errorf("json marshal: %w", err),
		)
		return err
	}

	res, err := client.http.R().
		SetContext(ctx).
		SetHeader("content-type", "application/json").
		SetBody(body).
		Post("https://mobile.powerschool.com/v3.0/graphql")
	if err != nil {
		client.tel.ReportBroken(
			report_client_graphql_query,
			fmt.Errorf("fetch: %w", err),
		)
		return err
	}

	parsed := graphqlResponse[O]{}
	err = json.Unmarshal(res.Body(), &parsed)
	if err != nil {
		client.tel.ReportBroken(
			report_client_graphql_query,
			fmt.Errorf("unmarshal json: %w", err),
		)
		return err
	}

	*output = parsed.Data

	client.tel.ReportDebug(
		fmt.Sprintf("%s response", report_client_graphql_query),
		name,
		parsed.Data,
	)

	return nil
}
