package telemetry

import (
	"fmt"
	"log/slog"
	"sync/atomic"
)

// SlogAPI implements API using the log/slog package.
type SlogAPI struct {
	idcounter *uint64
}

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

func (s SlogAPI) ReportDebug(message string, params ...any) {
	remainingPairs := []any{}
	s.formatParams(&remainingPairs, params)
	slog.Debug(message, remainingPairs...)
}

func (s SlogAPI) ReportCount(id string, count int64) {
	slog.Info("count", "id", id, "n", count)
}

func (s SlogAPI) StoreLongMessage(message string) (id string) {
	if s.idcounter == nil {
		var idcounter uint64
		s.idcounter = &idcounter
	}

	idNo := atomic.AddUint64(s.idcounter, 1)
	slog.Debug(message, "id", idNo)
	return fmt.Sprint(id)
}
