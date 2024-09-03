package restyutil

import (
	"log/slog"
	"os"
	"path/filepath"
	devenv "vcassist-backend/dev/env"
)

type FilesystemOutput struct {
	directory string
}

func NewFilesystemOutput(dir string) FilesystemOutput {
	dir, err := devenv.ResolvePath(dir)
	if err != nil {
		panic(err)
	}
	os.RemoveAll(dir)
	err = os.MkdirAll(dir, 0777)
	if err != nil {
		panic(err)
	}
	return FilesystemOutput{directory: dir}
}

func (o FilesystemOutput) Write(id string, contents string) {
	err := os.WriteFile(filepath.Join(o.directory, id), []byte(contents), 0600)
	if err != nil {
		slog.Warn("failed to write message info file", "id", id, "err", err)
	}
}
