package timezone

import "time"

var Location *time.Location

func init() {
	var err error
	Location, err = time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic(err)
	}
}

// force timezone to be in LA because sometimes our servers
// end up on east coast which will cause disturbances when
// manipulating dates based on <time.Time>.Year()/Month()/Day()/Hour()/...
func Now() time.Time {
	return time.Now().In(Location)
}
