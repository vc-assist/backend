package main

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"vcassist-backend/internal/components/db"
	"vcassist-backend/pkg/migrations"
)

func MustFindEnv(t *testing.T, variable string) string {
	res, ok := os.LookupEnv(variable)
	if !ok {
		t.Fatal(fmt.Sprintf("env var '%s' must be set to run this test", variable))
	}
	return res
}

func OpenInMemoryDB(t *testing.T) *sql.DB {
	dbtx, err := migrations.OpenDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = dbtx.Exec(db.Schema)
	if err != nil {
		t.Fatal(err)
	}
	return dbtx
}

func runTests(name string, body func(t *testing.T)) {
	testing.Main(
		func(pat, str string) (bool, error) {
			return true, nil
		},
		[]testing.InternalTest{
			{name, body},
		},
		nil,
		nil,
	)
}
