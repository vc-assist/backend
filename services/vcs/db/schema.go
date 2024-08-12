package db

import (
	_ "embed"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var Schema string
