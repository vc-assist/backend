package vcsmoodle

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"
	"vcassist-backend/lib/timezone"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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

func resolveMonthDay(month time.Month, day int) time.Time {
	now := timezone.Now()
	return time.Date(now.Year(), month, day, 0, 0, 0, 0, timezone.Location)
}

var monthDayRegex = regexp.MustCompile(`([A-Za-z]{3,9}) *(\d{1,2})`)
var monthDayDayRegex = regexp.MustCompile(`(\w+) *(\d{1,2}) *[^\d\w\s] *(\d{1,2})(?:[^\d]|$)`)

func parseTOCDate(ctx context.Context, text string) ([]time.Time, error) {
	ctx, span := tracer.Start(ctx, "parseTOCDate")
	defer span.End()

	span.SetAttributes(attribute.String("text", text))

	monthDayDayMatch := monthDayDayRegex.FindStringSubmatch(text)
	if len(monthDayDayMatch) >= 4 {
		month := parseMonth(monthDayDayMatch[1])
		day1, err := strconv.ParseInt(monthDayDayMatch[2], 10, 32)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		day2, err := strconv.ParseInt(monthDayDayMatch[3], 10, 32)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		span.SetAttributes(
			attribute.String("month", month.String()),
			attribute.Int64("day1", day1),
			attribute.Int64("day2", day2),
		)

		return []time.Time{
			resolveMonthDay(month, int(day1)),
			resolveMonthDay(month, int(day2)),
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
			span.RecordError(err)
			continue
		}
		dates = append(dates, resolveMonthDay(month, int(day)))
	}
	return dates, nil
}
