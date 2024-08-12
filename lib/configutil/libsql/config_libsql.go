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
		isNewDb := os.IsNotExist(statErr)
		if isNewDb {
			f, err := os.Create(dbpath)
			if err != nil {
				return nil, err
			}
			f.Close()
		}

		db, err := sql.Open("libsql", fmt.Sprintf("file:%s", dbpath))
		if err != nil {
			return nil, err
		}

		if isNewDb && schema != "" {
			_, err := db.Exec(schema)
			if err != nil {
				return nil, err
			}
		}

		return db, nil
	}

	urlQuery := ""
	if config.AuthToken != "" {
		values := url.Values{"authToken": []string{config.AuthToken}}
		urlQuery = "?" + values.Encode()
	}

	db, err := sql.Open("libsql", config.Url+urlQuery)
	if err != nil {
		return nil, err
	}
	return db, nil
}
