package telemetry

import (
	"fmt"
	"log/slog"
)

// SlogAPI implements API using the log/slog package.
type SlogAPI struct{}

func (SlogAPI) formatParams(out *[]any, params []any) {
	for i, p := range params {
		*out = append(
			*out,
			fmt.Sprintf("params.%d", i),
			p,
		)
	}
}

func (s SlogAPI) ReportBroken(id string, params ...any) {
	remainingPairs := []any{"id", id}
	s.formatParams(&remainingPairs, params)
	slog.Error("broken component", remainingPairs...)
}

func (s SlogAPI) ReportWarning(id string, params ...any) {
	remainingPairs := []any{"id", id}
	s.formatParams(&remainingPairs, params)
	slog.Warn("warning", remainingPairs...)
}

func (s SlogAPI) ReportCount(id string, count int64) {
	slog.Info("count", "id", id, "n", count)
}
