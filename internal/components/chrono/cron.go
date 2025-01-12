package chrono

import (
	"fmt"
	"vcassist-backend/internal/components/telemetry"

	"github.com/robfig/cron/v3"
)

// CronAPI is the interface that anything depending on things to happen on a cron job should use.
type CronAPI interface {
	Cron(spec string, callback func()) error
}

// StandardCron is the standard implementation of CronAPI using `github.com/robfig/cron/v3`
type StandardCron struct {
	cron *cron.Cron
}

// NewStandardCron is the constructor of StandardCron.
func NewStandardCron(tel telemetry.API) StandardCron {
	cronner := cron.New(
		cron.WithLogger(cronLogger{tel: tel}),
		cron.WithLocation(la),
	)
	cronner.Start()

	return StandardCron{
		cron: cronner,
	}
}

func (s StandardCron) Cron(spec string, callback func()) error {
	_, err := s.cron.AddFunc(spec, callback)
	return err
}

type cronLogger struct {
	tel telemetry.API
}

func (l cronLogger) formatParams(keysAndValues []any) []any {
	params := []any{}
	for i := 0; i < len(keysAndValues)/2; i++ {
		idx := i * 2
		key := keysAndValues[idx]
		value := keysAndValues[idx+1]
		params = append(params, fmt.Sprintf("%v: %v", key, value))
	}
	return params
}

func (l cronLogger) Info(msg string, keysAndValues ...any) {
	l.tel.ReportDebug(
		fmt.Sprintf("cron: %s", msg),
		l.formatParams(keysAndValues),
	)
}

func (l cronLogger) Error(err error, msg string, keysAndValues ...any) {
	l.tel.ReportBroken(
		"cron",
		fmt.Errorf("%s: %w", msg, err),
		l.formatParams(keysAndValues),
	)
}
