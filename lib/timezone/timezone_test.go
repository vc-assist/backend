package timezone

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCurrentWeek(t *testing.T) {
	loc, err := time.LoadLocation("Local")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		now         time.Time
		expectStart time.Time
		expectStop  time.Time
	}{
		{
			now:         time.Date(2024, time.August, 26, 0, 0, 0, 0, loc),
			expectStart: time.Date(2024, time.August, 25, 0, 0, 0, 0, loc),
			expectStop:  time.Date(2024, time.August, 31, 0, 0, 0, 0, loc),
		},
		{
			now:         time.Date(2024, time.August, 25, 0, 0, 0, 0, loc),
			expectStart: time.Date(2024, time.August, 25, 0, 0, 0, 0, loc),
			expectStop:  time.Date(2024, time.August, 31, 0, 0, 0, 0, loc),
		},
		{
			now:         time.Date(2024, time.August, 31, 0, 0, 0, 0, loc),
			expectStart: time.Date(2024, time.August, 25, 0, 0, 0, 0, loc),
			expectStop:  time.Date(2024, time.August, 31, 0, 0, 0, 0, loc),
		},
		{
			now:         time.Date(2024, time.September, 3, 0, 0, 0, 0, loc),
			expectStart: time.Date(2024, time.September, 1, 0, 0, 0, 0, loc),
			expectStop:  time.Date(2024, time.September, 7, 0, 0, 0, 0, loc),
		},
	}

	for _, test := range cases {
		start, stop := GetCurrentWeek(test.now)
		require.Equal(t, test.expectStart, start)
		require.Equal(t, test.expectStop, stop)
	}
}
