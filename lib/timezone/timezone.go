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

// gets the start and end of the current week
func GetCurrentWeek(now time.Time) (start time.Time, stop time.Time) {
	start = now.Add(-time.Hour * 24 * time.Duration(now.Weekday()))
	stop = now.Add(time.Hour * 24 * time.Duration(time.Saturday-now.Weekday()))
	return start, stop
}
