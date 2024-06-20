package vcsnet

import (
	"context"
	"fmt"
	"testing"
	"time"
	"vcassist-backend/lib/telemetry"

	"github.com/stretchr/testify/require"
)

func setupTelemetry(t testing.TB) func(testing.TB) {
	tel, err := telemetry.SetupFromEnv(context.Background(), "test:vcsnet")
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
	teardown := setupTelemetry(t)
	defer teardown(t)

	tz, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}

	events, err := FetchEvents(context.Background(), tz)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(events)

	require.Greater(t, len(events), 0)
}

func TestGetSchoolYear(t *testing.T) {
	tz, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}

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
