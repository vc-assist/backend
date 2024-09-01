package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	authdb "vcassist-backend/services/auth/db"
	keychaindb "vcassist-backend/services/keychain/db"
	linkerdb "vcassist-backend/services/linker/db"
	vcmoodledb "vcassist-backend/services/vcmoodle/db"
	vcsisdb "vcassist-backend/services/vcsis/db"

	_ "modernc.org/sqlite"
)

func cmd(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fullCmd := name
	for _, a := range args {
		fullCmd += " "
		fullCmd += a
	}

	fmt.Printf("$ %s\n", fullCmd)
	err := cmd.Run()
	if err != nil {
		os.Exit(1)
	}
}

func CreateLocalStack() error {
	err := os.Chdir("dev/local_stack")
	if err != nil {
		return err
	}
	cmd("docker", "compose", "up", "-d")
	return os.Chdir("../..")
}

func createDb(path string, schemas ...string) error {
	_, err := os.Stat(path)
	if err == nil {
		slog.Info("database already exists", "path", path)
		return nil
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()

	for _, s := range schemas {
		_, err := db.Exec(s)
		if err != nil {
			return err
		}
	}
	return nil
}

func CreateDevDatabases() error {
	var errs []error
	var err error

	err = createDb("dev/.state/auth_service.db", authdb.Schema)
	errs = append(errs, err)
	err = createDb("dev/.state/keychain_service.db", keychaindb.Schema)
	errs = append(errs, err)
	err = createDb("dev/.state/linker_service.db", linkerdb.Schema)
	errs = append(errs, err)
	err = createDb("dev/.state/vcsis_service.db", vcsisdb.Schema)
	errs = append(errs, err)
	err = createDb("dev/.state/vcmoodle_service.db", vcmoodledb.Schema)
	errs = append(errs, err)

	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

func PrintConfigLocations() {
	slog.Info("some tests will require you to create config files in dev/.state/... in order to run properly, please look at the result of skipped tests in `go test -v` to understand where to write the files.")
}
