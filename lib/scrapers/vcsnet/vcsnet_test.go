package vcsnet

import (
	"context"
	"testing"
	"time"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/lib/timezone"

	"github.com/stretchr/testify/require"
)

func setupTelemetry(t testing.TB) func(testing.TB) {
	tel, err := telemetry.SetupFromEnv(context.Background(), "test:scrapers/vcsnet")
	if err != nil {
		t.Fatal(err)
	}

	return func(t testing.TB) {
		err := tel.Shutdown(context.Background())
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestFetchEvents(t *testing.T) {
	cleanup := telemetry.SetupForTesting(t, "test:scrapers/vcsnet")
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	events, err := FetchEvents(ctx, timezone.Location)
	if err != nil {
		t.Fatal(err)
	}

	require.Greater(t, len(events), 0)
}

func TestGetSchoolYear(t *testing.T) {
	tz := timezone.Location

	testCases := []struct {
		now      time.Time
		expected SchoolYear
	}{
		{
			now: time.Date(2000, 5, 22, 0, 0, 0, 0, tz),
			expected: SchoolYear{
				StartYear: 1999,
				EndYear:   2000,
				StartTime: time.Date(1999, 8, 1, 0, 0, 0, 0, tz),
			},
		},
		{
			now: time.Date(2011, 12, 22, 0, 0, 0, 0, tz),
			expected: SchoolYear{
				StartYear: 2011,
				EndYear:   2012,
				StartTime: time.Date(2011, 8, 1, 0, 0, 0, 0, tz),
			},
		},
		{
			now: time.Date(2020, 6, 10, 0, 0, 0, 0, tz),
			expected: SchoolYear{
				StartYear: 2019,
				EndYear:   2020,
				StartTime: time.Date(2019, 8, 1, 0, 0, 0, 0, tz),
			},
		},
	}

	for _, test := range testCases {
		year := GetSchoolYear(test.now, tz)
		require.Equal(t, test.expected.StartYear, year.StartYear)
		require.Equal(t, test.expected.EndYear, year.EndYear)
		require.Equal(t, test.expected.StartTime, year.StartTime)
	}
}
