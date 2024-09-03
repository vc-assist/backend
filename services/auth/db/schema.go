package db

import (
	_ "embed"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var Schema string
