package telemetry

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

// SlogAPI implements API using the log/slog package.
type SlogAPI struct {
	logger    *slog.Logger
	idcounter *uint64
}

func NewSlogAPI(level slog.Level) SlogAPI {
	logger := slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level: level,
	}))

	var idcounter uint64
	return SlogAPI{
		idcounter: &idcounter,
		logger:    logger,
	}
}

func (s SlogAPI) Logger() *slog.Logger {
	return s.logger
}

func (SlogAPI) formatParams(out *[]any, params []any) {
	for i, p := range params {
		kv, isKv := p.(KV)
		if isKv {
			*out = append(*out, kv.Key, kv.Value)
			continue
		}
		*out = append(
			*out,
			fmt.Sprintf("param.%d", i),
			p,
		)
	}
}

func (s SlogAPI) ReportBroken(id string, params ...any) {
	remainingPairs := []any{}
	s.formatParams(&remainingPairs, params)
	s.logger.Error(id, remainingPairs...)
}

func (s SlogAPI) ReportWarning(id string, params ...any) {
	remainingPairs := []any{}
	s.formatParams(&remainingPairs, params)
	s.logger.Warn(id, remainingPairs...)
}

func (s SlogAPI) ReportDebug(message string, params ...any) {
	remainingPairs := []any{}
	s.formatParams(&remainingPairs, params)
	s.logger.Debug(message, remainingPairs...)
}

func (s SlogAPI) ReportCount(id string, count int64) {
	s.logger.Info("count", "id", id, "n", count)
}
