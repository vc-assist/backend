package devenv

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

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

func ResolvePath(path string) (string, error) {
	if strings.HasPrefix(path, "<dev_state>") {
		root, err := GetWorkspaceRoot()
		if err != nil {
			return "", err
		}

		subpath := strings.Replace(path, "<dev_state>/", "", 1)
		statepath := filepath.Join(root, "dev/.state", subpath)

		return statepath, nil
	}
	return path, nil
}
