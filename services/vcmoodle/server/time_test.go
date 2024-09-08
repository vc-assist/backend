package server

import (
	"testing"
	"time"
	"vcassist-backend/lib/telemetry"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type MonthDay struct {
	Month time.Month
	Day   int
}

func TestParseTOCDate(t *testing.T) {
	cleanup := telemetry.SetupForTesting("test:services/vcmoodle")
	defer cleanup()

	testCases := []struct {
		text     string
		expected []MonthDay
	}{
		{
			text: "Mar 16",
			expected: []MonthDay{
				{
					Month: time.March,
					Day:   16,
				},
			},
		},
		{
			text: "Mar 16/17",
			expected: []MonthDay{
				{
					Month: time.March,
					Day:   16,
				},
				{
					Month: time.March,
					Day:   17,
				},
			},
		},
		{
			text: "Mar 31/Apr 1",
			expected: []MonthDay{
				{
					Month: time.March,
					Day:   31,
				},
				{
					Month: time.April,
					Day:   1,
				},
			},
		},
		{
			text: "Mar 31/Apr 1/Apr 2",
			expected: []MonthDay{
				{
					Month: time.March,
					Day:   31,
				},
				{
					Month: time.April,
					Day:   1,
				},
				{
					Month: time.April,
					Day:   2,
				},
			},
		},
		{
			text: "Mar 31 (A)/Apr 1 (B)",
			expected: []MonthDay{
				{
					Month: time.March,
					Day:   31,
				},
				{
					Month: time.April,
					Day:   1,
				},
			},
		},
		{
			text: "1. January 8/9: Beginning of Quarter 3 and All That Glitters",
			expected: []MonthDay{
				{
					Month: time.January,
					Day:   8,
				},
				{
					Month: time.January,
					Day:   9,
				},
			},
		},
		{
			text: "9. January 31/February 1: In-class essay/Let's Write!",
			expected: []MonthDay{
				{
					Month: time.January,
					Day:   31,
				},
				{
					Month: time.February,
					Day:   1,
				},
			},
		},
		{
			text: "August 15, 2024",
			expected: []MonthDay{
				{
					Month: time.August,
					Day:   15,
				},
			},
		},
	}

	for _, test := range testCases {
		dates, err := parseTOCDate(test.text)
		if err != nil {
			t.Fatal(err)
		}

		dateMonthDays := make([]MonthDay, len(dates))
		for i, d := range dates {
			dateMonthDays[i] = MonthDay{
				Month: d.Month(),
				Day:   d.Day(),
			}
		}

		diff := cmp.Diff(
			test.expected, dateMonthDays,
			cmpopts.SortSlices(func(a, b MonthDay) bool {
				if a.Month == b.Month {
					return a.Day < b.Day
				}
				return a.Month < b.Month
			}),
		)
		if diff != "" {
			t.Fatal(diff)
		}
	}
}
