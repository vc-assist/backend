package chrono

import "time"

type API interface {
	Now() time.Time
	Location() *time.Location
}

type StandardImpl struct {
	location *time.Location
}

func NewStandardImpl() (StandardImpl, error) {
	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		return StandardImpl{}, err
	}
	return StandardImpl{location: location}, nil
}

func (s StandardImpl) Now() time.Time {
	return s.Now().In(s.location)
}

func (s StandardImpl) Location() *time.Location {
	return s.location
}
