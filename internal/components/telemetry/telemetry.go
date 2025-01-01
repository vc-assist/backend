package telemetry

import (
	"fmt"
)

// API is an abstraction over logging/metrics.
// This allows for assertions and tests for working logging/metrics to exist.
//
// note: fault injection point
type API interface {
	// ReportBroken reports a component that has broken in a way that should be addressed.
	//
	// The `id` is an fully qualified identifier that should indicate what **component** broke, not what specific piece
	// of the implementation of a component broke. To evaluate the "correctness" of an id, think of if you came across the
	// report in an admin dashboard in production, would you be able to find the place that is broken?
	//
	// ex. Suppose an HTTP request fails in a powerschool scraping component with a method called `GetCourses`.
	// You want to report the breakage, the id should be `powerschool.get-courses`, no more granular than that.
	// If you need to disambiguate and specify that it was HTTP that failed, then you can do that by adding a param
	// or wrapping the error with fmt.Errorf
	//
	// For more examples, do take a look at the `report_...` string constants scattered around various packages.
	//
	// Formatting rules:
	// 1) all lowercase
	// 2) use underscores for large components
	// 3) use dashes for methods part of a larger component
	//
	// Note 1: You do not need to put the whole file path into the identifier (like `internal.scrapers.moodle.view`),
	// use of ScopedAPI usually helps enough to disambiguate things between packages so all you usually have to do
	// is put the `<name of struct or intf>.<method>` as the id.
	//
	// Note 2: Remember that an id is just a string that helps you locate where something is, it does not need to carry
	// additional semantic information like whether or not something broke or not, that is already given by whether or
	// not you called ReportBroken or ReportWarning. As such, ids like `db.broken-query` should actually just be
	// `db.query`.
	ReportBroken(id string, params ...any)

	// ReportWarning reports a scenario that does not necessarily indicate brokenness, but may be subject to investigation
	//
	// For what value to provide as `id` refer to ReportBroken.
	ReportWarning(id string, params ...any)

	// ReportDebug reports some debug information that will be ignored in production
	ReportDebug(msg string, params ...any)

	// ReportCount reports the current count of a specific event at the current time, these counts should
	// not be summed but interpreted as points of data over time.
	//
	// For what value to provide as `id` refer to ReportBroken.
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
	s.inner.ReportBroken(fmt.Sprintf("%s: %s", s.namespace, id), params...)
}

func (s ScopedAPI) ReportWarning(id string, params ...any) {
	s.inner.ReportWarning(fmt.Sprintf("%s: %s", s.namespace, id), params...)
}

func (s ScopedAPI) ReportDebug(msg string, params ...any) {
	s.inner.ReportDebug(fmt.Sprintf("%s: %s", s.namespace, msg), params...)
}

func (s ScopedAPI) ReportCount(id string, count int64) {
	s.inner.ReportCount(fmt.Sprintf("%s: %s", s.namespace, id), count)
}
