package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"vcassist-backend/lib/telemetry"

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

	err = os.MkdirAll("dev/.state", 0777)
	if err != nil && !os.IsExist(err) {
		return err
	}

	err = CreateLocalStack()
	if err != nil {
		return err
	}
	PrintDisclaimer()

	return nil
}

func main() {
	telemetry.InitSlog(true)

	recreate := flag.Bool("recreate", false, "recreate the dev environment from scratch")
	flag.Parse()

	err := create(*recreate)
	if err != nil {
		slog.Error("failed to create dev environment", "err", err.Error())
		os.Exit(1)
	}

	slog.Info("dev environment created sucessfully!")
}
