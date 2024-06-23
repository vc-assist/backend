package auth

import "vcassist-backend/lib/configuration"

type EmailConfig struct {
	Server   string `json:"server"`
	Port     int    `json:"port"`
	Address  string `json:"address"`
	Password string `json:"password"`
}

type Config struct {
	Libsql configuration.Libsql `json:"database"`
	Email  EmailConfig          `json:"email"`
}
