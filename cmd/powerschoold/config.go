package main

import (
	"vcassist-backend/lib/configuration"
)

type OAuthConfig struct {
	BaseLoginUrl string `json:"base_login_url"`
	RefreshUrl   string `json:"refresh_url"`
	ClientId     string `json:"client_id"`
}

type Config struct {
	BaseUrl string               `json:"base_url"`
	OAuth   OAuthConfig          `json:"oauth"`
	Libsql  configuration.Libsql `json:"database"`
}
