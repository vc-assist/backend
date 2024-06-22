package powerschoolapi

import (
	"database/sql"
	"fmt"
	"vcassist-backend/lib/platforms/powerschool"

	"github.com/tursodatabase/go-libsql"
)

type DatabaseConfig struct {
	File      string `json:"file"`
	Url       string `json:"url"`
	AuthToken string `json:"auth_token"`
}

type Config struct {
	BaseUrl  string                  `json:"base_url"`
	OAuth    powerschool.OAuthConfig `json:"oauth"`
	Database DatabaseConfig          `json:"database"`
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
