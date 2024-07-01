package main

import (
	"vcassist-backend/lib/configuration"
)

type OAuthConfig struct {
	BaseLoginUrl string `json:"base_login_url"`
	RefreshUrl   string `json:"refresh_url"`
	ClientId     string `json:"client_id"`
}

type DatabaseConfig struct {
	Self configuration.Libsql `json:"self"`
	Auth configuration.Libsql `json:"auth"`
}

type Config struct {
	BaseUrl  string         `json:"base_url"`
	OAuth    OAuthConfig    `json:"oauth"`
	Database DatabaseConfig `json:"database"`
}
