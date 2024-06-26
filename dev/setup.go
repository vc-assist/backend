package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	devenv "vcassist-backend/dev/env"

	"github.com/tcnksm/go-input"
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

func CreatePowerschoolApiDevDB() error {
	db, err := sql.Open("sqlite", "cmd/powerschool_api/state.db")
	if err != nil {
		return err
	}
	defer db.Close()
	schema, err := os.ReadFile("cmd/powerschool_api/db/schema.sql")
	if err != nil {
		return err
	}
	_, err = db.Exec(string(schema))
	if strings.Contains(err.Error(), "already exists") {
		return nil
	}
	return err
}

func CreateLocalStack() error {
	err := os.Chdir("dev/local_stack")
	if err != nil {
		return err
	}
	cmd("docker", "compose", "up", "-d")
	return os.Chdir("../..")
}

func SetupMoodleTests() error {
	_, err := os.Stat("dev/.state/moodle_credentials.json")
	if !os.IsNotExist(err) {
		slog.Info("moodle credentials have already been provided")
		return err
	}
	ui := input.DefaultUI()

	opts := &input.Options{
		Default: "",
		Mask:    false,
		Loop:    true,
	}
	baseUrl, err := ui.Ask("moodle tests' base url:", opts)
	if err != nil {
		return err
	}
	username, err := ui.Ask("moodle tests' username:", opts)
	if err != nil {
		return err
	}
	password, err := ui.Ask("moodle tests' password:", opts)
	if err != nil {
		return err
	}
	specificCourse, err := ui.Ask("specific course (lowercase name) to target in moodle tests:", opts)
	if err != nil {
		return err
	}

	config := devenv.MoodleTestConfig{
		BaseUrl:        baseUrl,
		Username:       username,
		Password:       password,
		SpecificCourse: specificCourse,
	}
	cached, err := json.Marshal(config)
	if err != nil {
		return err
	}
	err = os.WriteFile("dev/.state/moodle_credentials.json", cached, 0777)
	if err != nil {
		return err
	}
	return nil
}
