package main

import "vcassist-backend/lib/configuration"

type EmailConfig struct {
	Server       string `json:"server"`
	Port         int    `json:"port"`
	EmailAddress string `json:"email_address"`
	Password     string `json:"password"`
}

type AuthConfig struct {
	Email  EmailConfig          `json:"email"`
	Libsql configuration.Libsql `json:"database"`
}
