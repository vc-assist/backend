package migrations

import (
	"database/sql"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func openSqlite(path string) (*sql.DB, error) {
	os.MkdirAll(filepath.Dir(path), 0777)

	db, err := sql.Open("sqlite", path)
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

func OpenDB(schema, path string) (*sql.DB, error) {
	db, err := openSqlite(path)
	if err != nil {
		return nil, err
	}

	_, err = exec.LookPath("atlas")
	if os.IsNotExist(err) {
		slog.Warn(
			"could not find 'atlas' executable on path, is it installed? skipping migrations...",
			"path", path,
		)
		return db, nil
	}

	slog.Info("running migrations on db", "path", path)

	err = db.Close()
	if err != nil {
		return nil, err
	}
	err = os.WriteFile("temp_migration_schema.sql", []byte(schema), 0666)
	if err != nil {
		return nil, err
	}

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
		slog.Warn("failed to run migrations", "err", err)
	}

	err = os.Remove("temp_migration_schema.sql")
	if err != nil {
		slog.Warn("could not delete temp_migration_schema.sql", "err", err)
	}

	return openSqlite(path)
}
