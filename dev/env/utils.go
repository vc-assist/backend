package devenv

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"vcassist-backend/lib/configutil"
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

func GetStateFilePath(path string) (string, error) {
	root, err := GetWorkspaceRoot()
	if err != nil {
		return "", err
	}
	configPath := filepath.Join(root, "dev/.state", path)
	return configPath, nil
}

func GetStateFile(path string) ([]byte, error) {
	configPath, err := GetStateFilePath(path)
	if err != nil {
		return nil, err
	}
	contents, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("no file at %s", configPath)
	}
	return contents, err
}

func GetStateConfig[T any](path string) (T, error) {
	configPath, err := GetStateFilePath(path)
	if err != nil {
		var out T
		return out, err
	}
	out, err := configutil.ReadConfig[T](configPath)
	return out, err
}

func ResolvePath(path string) (string, error) {
	if !strings.HasPrefix(path, "<dev_state>") {
		return path, nil
	}

	root, err := GetWorkspaceRoot()
	if err != nil {
		return "", err
	}

	err = os.Mkdir(filepath.Join(root, "dev", ".state"), 0777)
	if !os.IsExist(err) && err != nil {
		return "", err
	}

	subpath := filepath.Join(strings.Split(path, string(os.PathSeparator))[1:]...)
	statepath := filepath.Join(
		root, "dev", ".state", subpath,
	)

	return statepath, nil
}
