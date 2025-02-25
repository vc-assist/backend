package chrono

import (
	"time"
)

var la *time.Location

func init() {
	var err error
	la, err = time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic(err)
	}
}

// LA returns a [*time.Location] for America/Los_Angeles
func LA() *time.Location {
	return la
}

// TimeAPI is the interface that anything depending on the system clock should use.
type TimeAPI interface {
	// Now returns the current time, the timezone of the time will default to America/Los_Angeles.
	Now() time.Time
}

// StandardTime is the standard implementation of TimeAPI using the standard library.
type StandardTime struct{}

// NewStandardTime is the constructor of StandardTime.
func NewStandardTime() StandardTime {
	return StandardTime{}
}

func (s StandardTime) Now() time.Time {
	return s.Now().In(la)
}
