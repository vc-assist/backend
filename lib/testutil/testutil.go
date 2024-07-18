package testutil

import (
	"database/sql"
	"fmt"
	"testing"
	"vcassist-backend/lib/telemetry"

	_ "modernc.org/sqlite"
)

type ServiceParams struct {
	Name string
	// if unspecified, it will skip setting up a db
	DbSchema string
	// if unspecified, it will use `:memory:`
	DbPath string
}

type ServiceResult struct {
	DB *sql.DB
}

func SetupService(t testing.TB, params ServiceParams) (ServiceResult, func()) {
	cleanup := telemetry.SetupForTesting(t, fmt.Sprintf("test:%s", params.Name))

	dbpath := ":memory:"
	if params.DbPath != "" {
		dbpath = params.DbPath
	}
	sqlite, err := sql.Open("sqlite", dbpath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlite.Exec(params.DbSchema)
	if err != nil {
		t.Fatal(err)
	}

	return ServiceResult{
		DB: sqlite,
	}, cleanup
}
