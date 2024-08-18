package utils

import (
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
)

func NewTable() table.Writer {
	t := table.NewWriter()
	t.SetStyle(table.StyleRounded)
	t.SetOutputMirror(os.Stdout)
	return t
}
