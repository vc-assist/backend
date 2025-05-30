package fuzzing

import (
	"math/rand"
	"time"
	testutil "vcassist-backend/test/util"
)

type timeShim struct {
	current   time.Time
	increment func(rndm *rand.Rand) int
	rndm      *rand.Rand
}

func newTimeShim(rndm *rand.Rand) *timeShim {
	return &timeShim{
		current: time.Now(),
		// 0: add anywhere from (0, 60) seconds
		// 1: add anywhere from (0, 60) minutes
		// 2: add anywhere from (0, 24) hours
		// 3: add anywhere from (0, 30) days
		increment: testutil.RandomSwitch(3, 3, 3, 1),
		rndm:      rndm,
	}
}

func (s timeShim) randDuration() time.Duration {
	var dur time.Duration
	switch s.increment(s.rndm) {
	case 0:
		dur = time.Duration(s.rndm.Intn(59)+1) * time.Second
	case 1:
		dur = time.Duration(s.rndm.Intn(59)+1) * time.Minute
	case 2:
		dur = time.Duration(s.rndm.Intn(23)+1) * time.Hour
	case 3:
		dur = time.Duration(s.rndm.Intn(29)+1) * time.Hour * 24
	}
	return dur
}

func (s timeShim) Now() time.Time {
	return s.current
}
