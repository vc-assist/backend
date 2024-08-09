package configlibsql

import (
	"database/sql"
	"fmt"
	devenv "vcassist-backend/dev/env"

	"github.com/tursodatabase/go-libsql"
)

type Struct struct {
	File      string `json:"file"`
	Url       string `json:"url"`
	AuthToken string `json:"auth_token"`
}

func (config Struct) OpenDB() (*sql.DB, error) {
	if config.Url == "" {
		if config.File == "" {
			return nil, fmt.Errorf("a subpath was not specified")
		}
		dbpath, err := devenv.ResolvePath(config.File)
		if err != nil {
			return nil, err
		}
		return sql.Open("libsql", fmt.Sprintf("file:%s", dbpath))
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
