package devenv

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	"github.com/tcnksm/go-input"
)

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

type MoodleTestConfig struct {
	BaseUrl  string `json:"base_url"`
	Username string `json:"username"`
	Password string `json:"password"`
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
	baseUrl, err := ui.Ask("moodle base url:", opts)
	if err != nil {
		return err
	}
	username, err := ui.Ask("moodle username:", opts)
	if err != nil {
		return err
	}
	password, err := ui.Ask("moodle password:", opts)
	if err != nil {
		return err
	}

	config := MoodleTestConfig{
		BaseUrl:  baseUrl,
		Username: username,
		Password: password,
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
