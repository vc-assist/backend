package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
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

func CreateLocalStack() error {
	err := os.Chdir("dev/local_stack")
	if err != nil {
		return err
	}
	cmd("docker", "compose", "up", "-d")
	return os.Chdir("../..")
}

func SetupMoodleTests() error {
	_, err := os.Stat("dev/.state/moodle_config.json5")
	if !os.IsNotExist(err) {
		slog.Info("moodle credentials have already been set")
		return err
	}
	ui := input.DefaultUI()

	opts := &input.Options{
		Default: "",
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
	password, err := ui.Ask("moodle tests' password:", &input.Options{
		Default: "",
		Mask:    true,
		Loop:    true,
	})
	if err != nil {
		return err
	}
	viewSpecificCourse, err := ui.Ask("the course (make sure it's in the format `<course name> - <teacher name>` as seen on the moodle website) to target in the moodle/view tests:", opts)
	if err != nil {
		return err
	}
	editSpecificCourse, err := ui.Ask("the course (make sure it's in the format `<course name> - <teacher name>` as seen on the moodle website) to target in the moodle/edit tests:", opts)
	if err != nil {
		return err
	}

	config := devenv.MoodleTestConfig{
		BaseUrl:  baseUrl,
		Username: username,
		Password: password,
		ViewConfig: devenv.ViewMoodleTestConfig{
			TargetCourse: viewSpecificCourse,
		},
		EditConfig: devenv.EditMoodleTestConfig{
			TargetCourse: editSpecificCourse,
		},
	}
	cached, err := json.Marshal(config)
	if err != nil {
		return err
	}
	err = os.WriteFile("dev/.state/moodle_config.json5", cached, 0777)
	if err != nil {
		return err
	}

	slog.Info("moodle test configuration written to `dev/.state/moodle_config.json5`, make sure you check it to ensure it's correct.")

	return nil
}
