package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	setup "vcassist-backend/dev/setup"

	_ "modernc.org/sqlite"
)

func create(recreate bool) error {
	_, err := os.Stat("go.mod")
	if os.IsNotExist(err) {
		return fmt.Errorf("the dev environment must be created in the repository root (the same directory as the 'go.mod' file)")
	}

	if recreate {
		err = os.RemoveAll("dev/.state")
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	err = setup.CreateLocalStack()
	if err != nil {
		return err
	}
	err = setup.CreatePowerschoolApiDevDB()
	if err != nil {
		return err
	}
	err = setup.SetupMoodleTests()
	if err != nil {
		return err
	}

	return nil
}

func main() {
	recreate := flag.Bool("recreate", false, "recreate the dev environment from scratch")
	flag.Parse()

	err := create(*recreate)
	if err != nil {
		slog.Error("failed to create dev environment", "err", err.Error())
		os.Exit(1)
	}

	slog.Info("dev environment created sucessfully!")
}
