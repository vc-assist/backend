package chrono

import (
	"fmt"
	"time"
	"vcassist-backend/internal/components/telemetry"

	"github.com/robfig/cron/v3"
)

type API interface {
	Now() time.Time
	Location() *time.Location
	Cron(spec string, callback func()) error
}

type StandardImpl struct {
	location *time.Location
	cron     *cron.Cron
}

func NewStandardImpl(tel telemetry.API) (StandardImpl, error) {
	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		return StandardImpl{}, err
	}

	cronner := cron.New(
		cron.WithLogger(cronLogger{tel: tel}),
		cron.WithLocation(location),
	)
	cronner.Start()

	return StandardImpl{
		location: location,
		cron:     cronner,
	}, nil
}

func (s StandardImpl) Now() time.Time {
	return s.Now().In(s.location)
}

func (s StandardImpl) Location() *time.Location {
	return s.location
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

func (s StandardImpl) Cron(spec string, callback func()) error {
	_, err := s.cron.AddFunc(spec, callback)
	return err
}
