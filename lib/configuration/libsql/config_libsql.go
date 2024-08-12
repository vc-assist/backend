package configlibsql

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	devenv "vcassist-backend/dev/env"

	"github.com/tursodatabase/go-libsql"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

// if only "file" is specified, it will open the file local-only mode
// if only "url" is specified, it will open the database in remote-only mode
// if "url" is specified alongside "file" it will open the database in embedded-replica mode
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

	if config.File == "" {
		values := url.Values{}
		if config.AuthToken != "" {
			values.Add("authToken", config.AuthToken)
		}
		db, err := sql.Open("libsql", config.Url+"?"+values.Encode())
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open db %s: %s", config.Url, err)
			os.Exit(1)
		}
		return db, nil
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
