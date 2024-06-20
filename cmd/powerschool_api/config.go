package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"vcassist-backend/lib/platforms/powerschool"
	"vcassist-backend/lib/telemetry"

	"github.com/titanous/json5"
	"github.com/tursodatabase/go-libsql"
)

type DatabaseConfig struct {
	File      string `json:"file"`
	Url       string `json:"url"`
	AuthToken string `json:"auth_token"`
}

type Config struct {
	BaseUrl   string                  `json:"base_url"`
	OAuth     powerschool.OAuthConfig `json:"oauth"`
	Database  DatabaseConfig          `json:"database"`
	Telemetry telemetry.Config        `json:"telemetry"`
}

func MustLoadConfig(ctx context.Context, path string) Config {
	cfgFile, err := os.ReadFile(path)
	if err != nil {
		slog.Error("failed to open config file", "err", err.Error())
		os.Exit(1)
	}

	config := Config{}
	err = json5.Unmarshal(cfgFile, &config)
	if err != nil {
		slog.Error("failed to parse config file", "err", err.Error())
		os.Exit(1)
	}

	return config
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
