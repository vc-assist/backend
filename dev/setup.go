package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"

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

func PrintDisclaimer() {
	slog.Info("some tests will require you to provide credentials in order to run, please look at the result of skipped tests in `go test -v` to understand where to write secret config files.")
}
