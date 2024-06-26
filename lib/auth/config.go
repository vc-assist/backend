package auth

import "vcassist-backend/lib/configuration"

type AuthConfig struct {
	Email    EmailConfig          `json:"email"`
	Database configuration.Libsql `json:"database"`
}

func ServiceFromConfig(c AuthConfig) (Service, error) {
	db, err := c.Database.OpenDB()
	if err != nil {
		return Service{}, err
	}
	return NewService(db, c.Email), nil
}

func ServiceFromEnv() (Service, error) {
	config, err := configuration.ReadRecursively[AuthConfig]("auth.json5")
	if err != nil {
		return Service{}, err
	}
	return ServiceFromConfig(config)
}

