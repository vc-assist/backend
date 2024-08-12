package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	devenv "vcassist-backend/dev/env"

	authdb "vcassist-backend/services/auth/db"
	gradesnapshotsdb "vcassist-backend/services/gradesnapshots/db"
	keychaindb "vcassist-backend/services/keychain/db"
	linkerdb "vcassist-backend/services/linker/db"
	powerservicedb "vcassist-backend/services/powerservice/db"
	vcsdb "vcassist-backend/services/vcs/db"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
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

func createDb(filename, schema string) error {
	authPath, err := devenv.ResolvePath(filepath.Join("<dev_state>", filename))
	if err != nil {
		return err
	}

	_, err = os.Stat(authPath)
	if err == nil {
		fmt.Println("database already created at", authPath)
		return nil
	}

	fmt.Println("creating database at", authPath)
	db, err := sql.Open("sqlite", authPath)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(schema)
	return err
}

func CreateEmptyServiceDBs() error {
	err := createDb("auth_service.db", authdb.Schema)
	if err != nil {
		return err
	}
	err = createDb("gradesnapshots_service.db", gradesnapshotsdb.Schema)
	if err != nil {
		return err
	}
	err = createDb("keychain_service.db", keychaindb.Schema)
	if err != nil {
		return err
	}
	err = createDb("linker_service.db", linkerdb.Schema)
	if err != nil {
		return err
	}
	err = createDb("powerservice_service.db", powerservicedb.Schema)
	if err != nil {
		return err
	}
	err = createDb("vcs_service.db", vcsdb.Schema)
	if err != nil {
		return err
	}
	return err
}

func PrintConfigLocations() {
	slog.Info("some tests will require you to create config files in .dev/state/... in order to run properly, please look at the result of skipped tests in `go test -v` to understand where to write the files.")
}
