package migrations

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func wrapOpenDB(err error) error {
	return fmt.Errorf("open db: %w", err)
}

func OpenDB(path string) (*sql.DB, error) {
	if path != ":memory:" {
		os.MkdirAll(filepath.Dir(path), 0777)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, wrapOpenDB(err)
	}

	// see this stackoverflow post for information on why the following
	// lines exist: https://stackoverflow.com/questions/35804884/sqlite-concurrent-writing-performance
	db.SetMaxOpenConns(1)
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		return nil, wrapOpenDB(err)
	}

	return db, nil
}

func wrapOpenAndMigrate(err error) error {
	return fmt.Errorf("open and migrate db: %w", err)
}

func OpenAndMigrateDB(schema, path string) (*sql.DB, error) {
	// to ensure that the db actually exists
	db, err := OpenDB(path)
	if err != nil {
		return nil, wrapOpenAndMigrate(err)
	}
	err = db.Close()
	if err != nil {
		return nil, wrapOpenAndMigrate(err)
	}

	_, err = exec.LookPath("atlas")
	if os.IsNotExist(err) {
		return db, wrapOpenAndMigrate(fmt.Errorf(
			"could not find 'atlas' executable on path, is it installed? skipping migrations...",
		))
	}

	err = os.WriteFile("temp_migration_schema.sql", []byte(schema), 0666)
	if err != nil {
		return nil, wrapOpenAndMigrate(err)
	}
	defer func() {
		err = os.Remove("temp_migration_schema.sql")
		if err != nil {
			slog.Warn("could not delete temp_migration_schema.sql", "err", err)
		}
	}()

	dbUrl := url.URL{
		Scheme: "sqlite",
		Path:   path,
	}
	cmd := exec.Command(
		"atlas", "schema", "apply",
		"--url", dbUrl.String(),
		"--to", "file://temp_migration_schema.sql",
		"--dev-url", "sqlite://file?mode=memory",
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return nil, wrapOpenAndMigrate(err)
	}

	return OpenDB(path)
}
