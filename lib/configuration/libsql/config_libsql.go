package configlibsql

import (
	"database/sql"
	"fmt"
	"net/url"
	devenv "vcassist-backend/dev/env"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
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

	values := url.Values{}
	if config.AuthToken != "" {
		values.Add("authToken", config.AuthToken)
	}
	db, err := sql.Open("libsql", config.Url+"?"+values.Encode())
	if err != nil {
		return nil, err
	}
	return db, nil
}
