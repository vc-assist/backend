package server

import (
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"
	"vcassist-backend/lib/timezone"
)

var referenceMonths = []string{
	"january",
	"february",
	"march",
	"april",
	"may",
	"june",
	"july",
	"august",
	"september",
	"october",
	"november",
	"december",
}

func parseMonth(text string) time.Month {
	text = strings.ToLower(text)
	for i, month := range referenceMonths {
		if strings.Contains(month, text) {
			return time.January + time.Month(i)
		}
	}
	return -1
}

func resolveTOCMonthDay(month time.Month, day int) time.Time {
	now := timezone.Now()
	year := now.Year()

	if (month >= time.August && month <= time.December) &&
		(now.Month() < time.June && now.Month() >= time.January) {
		year--
	}
	if (month >= time.January && month < time.June) &&
		(now.Month() >= time.August && now.Month() <= time.December) {
		year++
	}

	return time.Date(year, month, day, 0, 0, 0, 0, timezone.Location)
}

var monthDayRegex = regexp.MustCompile(`([A-Za-z]{3,9}) *(\d{1,2})`)
var monthDayDayRegex = regexp.MustCompile(`(\w+) *(\d{1,2}) *[^\d\w\s] *(\d{1,2})(?:[^\d]|$)`)

func parseTOCDate(text string) ([]time.Time, error) {
	monthDayDayMatch := monthDayDayRegex.FindStringSubmatch(text)
	if len(monthDayDayMatch) >= 4 {
		month := parseMonth(monthDayDayMatch[1])
		day1, err := strconv.ParseInt(monthDayDayMatch[2], 10, 32)
		if err != nil {
			return nil, err
		}
		day2, err := strconv.ParseInt(monthDayDayMatch[3], 10, 32)
		if err != nil {
			return nil, err
		}

		return []time.Time{
			resolveTOCMonthDay(month, int(day1)),
			resolveTOCMonthDay(month, int(day2)),
		}, nil
	}

	monthDayMatches := monthDayRegex.FindAllStringSubmatch(text, -1)
	var dates []time.Time
	for _, match := range monthDayMatches {
		if len(match) < 3 {
			continue
		}
		month := parseMonth(match[1])
		day, err := strconv.ParseInt(match[2], 10, 32)
		if err != nil {
			slog.Warn("failed to parse day", "matches", match, "err", err)
			continue
		}
		dates = append(dates, resolveTOCMonthDay(month, int(day)))
	}
	return dates, nil
}
