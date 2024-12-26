package telemetry

import (
	"fmt"
)

// API is an abstraction over logging/metrics.
// This allows for assertions and tests for working logging/metrics to exist.
//
// note: fault injection point
type API interface {
	// ReportBroken reports a component that has broken in a way that should be addressed
	ReportBroken(id string, params ...any)

	// ReportWarning reports a scenario that does not necessarily indicate brokenness, but may be subject to investigation
	ReportWarning(id string, params ...any)

	// ReportCount reports the current count of a specific event at the current time, these counts should
	// not be summed but interpreted as points of data over time.
	ReportCount(id string, count int64)
}

// ScopedAPI is a telemetry API that attaches a namespace for a given API, kind of like creating a
// "sub" logger using things like log.New(), in which you can define the prefix for the logs.
type ScopedAPI struct {
	namespace string
	inner     API
}

// NewScopedAPI creates a ScopedAPI out of a given namespace and another api.
func NewScopedAPI(namespace string, inner API) ScopedAPI {
	return ScopedAPI{namespace: namespace, inner: inner}
}

func (s ScopedAPI) ReportBroken(id string, params ...any) {
	s.inner.ReportBroken(fmt.Sprintf("%s:%s", s.namespace, id), params...)
}

func (s ScopedAPI) ReportWarning(id string, params ...any) {
	s.inner.ReportWarning(fmt.Sprintf("%s:%s", s.namespace, id), params...)
}

func (s ScopedAPI) ReportCount(id string, count int64) {
	s.inner.ReportCount(fmt.Sprintf("%s:%s", s.namespace, id), count)
}
