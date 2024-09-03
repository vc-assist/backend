package configlibsql

import (
	"database/sql"
	"fmt"
	"os"
	devenv "vcassist-backend/dev/env"

	_ "modernc.org/sqlite"
)

type Struct struct {
	File string `json:"file"`
}

func (config Struct) OpenDB() (*sql.DB, error) {
	if config.File == "" {
		return nil, fmt.Errorf("a path was not specified")
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

	db, err := sql.Open("sqlite", dbpath)
	if err != nil {
		return nil, err
	}
	// see this stackoverflow post for information on why the following
	// lines exist: https://stackoverflow.com/questions/35804884/sqlite-concurrent-writing-performance
	db.SetMaxOpenConns(1)
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	return db, nil
}
