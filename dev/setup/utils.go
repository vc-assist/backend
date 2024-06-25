package devenv

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

var modName = regexp.MustCompile(`(?m)^module *([\w\-_]+)$`)

func isWorkspaceRoot(currentdir string) bool {
	mod, err := os.ReadFile(filepath.Join(currentdir, "go.mod"))
	if err != nil {
		return false
	}
	matches := modName.FindSubmatch(mod)
	isRoot := len(matches) >= 2 && string(matches[1]) >= "vcassist-backend"
	return isRoot
}

func GetWorkspaceRoot() (string, error) {
	currentdir, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}
	root, err := filepath.Abs("/")
	if err != nil {
		return "", err
	}

	for currentdir != root {
		isRoot := isWorkspaceRoot(currentdir)
		if !isRoot {
			currentdir = filepath.Join(currentdir, "..")
			continue
		}
		return currentdir, nil
	}

	return "", os.ErrNotExist
}

func GetStateFile(path string) ([]byte, error) {
	root, err := GetWorkspaceRoot()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(root, "dev/.state", path)
	contents, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("no file at %s", configPath)
	}
	return contents, err
}
