package powerschoolapi

import (
	"database/sql"
	"fmt"

	"github.com/tursodatabase/go-libsql"
)

type DatabaseConfig struct {
	File      string `json:"file"`
	Url       string `json:"url"`
	AuthToken string `json:"auth_token"`
}

type OAuthConfig struct {
	BaseLoginUrl string `json:"base_login_url"`
	RefreshUrl   string `json:"refresh_url"`
	ClientId     string `json:"client_id"`
}

type Config struct {
	BaseUrl  string         `json:"base_url"`
	OAuth    OAuthConfig    `json:"oauth"`
	Database DatabaseConfig `json:"database"`
}

func OpenDB(config DatabaseConfig) (*sql.DB, error) {
	if config.Url == "" {
		return sql.Open("libsql", fmt.Sprintf("file:%s", config.File))
	}

	var opts []libsql.Option
	if config.AuthToken != "" {
		opts = []libsql.Option{
			libsql.WithAuthToken(config.AuthToken),
		}
	}

	connector, err := libsql.NewEmbeddedReplicaConnector(
		config.File,
		config.Url,
		opts...,
	)
	if err != nil {
		return nil, err
	}
	return sql.OpenDB(connector), nil
}
