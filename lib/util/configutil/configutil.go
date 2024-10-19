package configutil

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"dario.cat/mergo"
	"github.com/titanous/json5"
)

func splitExt(f string) (string, string) {
	for i := len(f) - 1; i >= 0; i-- {
		if f[i] == '.' {
			return f[0:i], f[i+1:]
		}
	}
	return f, ""
}

// reads a configuration file, `name` should come with a file extension,
// it will automatically be lopped off to produce the other extensions.
// this function will merge the following files, where higher number is more prioritized.
// 1. <name>.<ext>
// 2. <name>.local.<ext>
func ReadConfig[T any](name string) (T, error) {
	var out T
	allNotFound := true

	dirname := filepath.Dir(name)
	basename := filepath.Base(name)
	prefixname, ext := splitExt(basename)

	defaultFile, err := os.ReadFile(name)
	if err != nil && !os.IsNotExist(err) {
		return out, err
	}
	if len(defaultFile) > 0 {
		err = json5.Unmarshal(defaultFile, &out)
		if err != nil {
			return out, err
		}
		allNotFound = false
	}

	localFilepath := filepath.Join(
		dirname,
		fmt.Sprintf("%s.local.%s", prefixname, ext),
	)
	localFile, err := os.ReadFile(localFilepath)
	if err != nil && !os.IsNotExist(err) {
		return out, err
	}
	if len(localFile) > 0 {
		var override T
		err = json5.Unmarshal(localFile, &override)
		if err != nil {
			return out, err
		}
		err = mergo.Merge(&out, override, mergo.WithOverride)
		if err != nil {
			return out, err
		}
		slog.Info("merging config with local overrides", "local", localFilepath)
		allNotFound = false
	}

	if allNotFound {
		return out, os.ErrNotExist
	}

	return out, nil
}

// ReadConfig but it recursively goes up the filesystem until the root
// to find a configuration file matching the name.
func ReadRecursively[T any](name string) (T, error) {
	var defaultOut T

	root, err := filepath.Abs("/")
	if err != nil {
		return defaultOut, err
	}
	current, err := os.Getwd()
	if err != nil {
		return defaultOut, err
	}

	for current != root {
		config, err := ReadConfig[T](filepath.Join(current, name))
		if os.IsNotExist(err) {
			current = filepath.Join(current, "..")
			continue
		}
		if err != nil {
			return defaultOut, err
		}

		return config, nil
	}

	return defaultOut, os.ErrNotExist
}
