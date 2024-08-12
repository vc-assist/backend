package configlibsql

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	devenv "vcassist-backend/dev/env"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

type Struct struct {
	File      string `json:"file"`
	Url       string `json:"url"`
	AuthToken string `json:"auth_token"`
}

func (config Struct) OpenDB(schema string) (*sql.DB, error) {
	if config.Url == "" {
		if config.File == "" {
			return nil, fmt.Errorf("a subpath was not specified")
		}
		dbpath, statErr := devenv.ResolvePath(config.File)
		if statErr != nil {
			return nil, statErr
		}

		_, statErr = os.Stat(dbpath)

		db, err := sql.Open("libsql", fmt.Sprintf("file:%s", dbpath))
		if err != nil {
			return nil, statErr
		}

		if os.IsNotExist(statErr) && schema != "" {
			_, err := db.Exec(schema)
			if err != nil {
				return nil, err
			}
		}

		return db, nil
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
