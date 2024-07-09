package main

import (
	"vcassist-backend/lib/configuration"
	authd "vcassist-backend/services/auth"
)

type AuthConfig struct {
	Email  authd.EmailConfig    `json:"email"`
	Libsql configuration.Libsql `json:"database"`
}
