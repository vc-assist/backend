package db

import (
	_ "embed"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var Schema string

// this is a list of 2 element tuples
// the first element of the tuple is the time (it has type float64), but should be interpreted as int64
// the second element of the tuple is the value, a float64
type GetGradeSnapshotsRowGrades [][2]float64
