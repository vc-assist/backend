package vcsnet

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"time"
	"vcassist-backend/lib/timezone"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
)

type SchoolYear struct {
	StartYear int
	EndYear   int
	StartTime time.Time
}

// gets the current school year, or if on summer break,
// the previous school year
func GetSchoolYear(now time.Time, tz *time.Location) SchoolYear {
	year := now.Year()
	month := now.Month()

	// encompasses S1
	if month >= 8 {
		return SchoolYear{
			StartYear: year,
			EndYear:   year + 1,
			StartTime: time.Date(year, 8, 1, 0, 0, 0, 0, tz),
		}
	}

	// encompasses summer break & S2
	return SchoolYear{
		StartYear: year - 1,
		EndYear:   year,
		StartTime: time.Date(year-1, 8, 1, 0, 0, 0, 0, tz),
	}
}

func GetYearRange(now time.Time) (int, int) {
	year := now.Year()
	month := now.Month()
	if month >= 8 && month <= 12 {
		return year, year + 1
	}
	return year - 1, year
}

type Event struct {
	Name string
	Date time.Time
}

var client = resty.New()

func FetchEvents(ctx context.Context, tz *time.Location) ([]Event, error) {

	link, err := url.Parse("https://www.vcs.net/fs/elements/39337")
	if err != nil {
		return nil, err
	}

	schoolYear := GetSchoolYear(timezone.Now(), tz)

	slog.DebugContext(ctx, "event bounds", "start_year", schoolYear.StartYear, "end_year", schoolYear.EndYear)

	query := url.Values{}
	query.Add("start_date", fmt.Sprintf("%04d-08-01", schoolYear.StartYear))
	query.Add("end_date", fmt.Sprintf("%04d-08-01", schoolYear.EndYear))
	query.Add("keywords", "")
	query.Add("is_draft", "false")
	query.Add("is_load_more", "true")
	query.Add("parent_id", "39337")

	currentDate := schoolYear.StartTime

	query.Add("_", strconv.FormatInt(currentDate.Unix(), 10))

	var result []Event
	var errList []error
	resultLock := sync.Mutex{}
	wg := sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		currentDate = currentDate.AddDate(0, 1, 0)
		query.Set("cal_date", fmt.Sprintf(
			"%04d-%02d-%02d",
			currentDate.Year(),
			currentDate.Month(),
			currentDate.Day(),
		))
		link.RawQuery = query.Encode()

		wg.Add(1)
		go func() {
			defer wg.Done()

			events, err := parseCalendar(ctx, link.String(), tz)
			if err != nil {
				slog.ErrorContext(ctx, "failed to parse calendar page", "err", err)
				errList = append(errList, err)
				return
			}

			resultLock.Lock()
			defer resultLock.Unlock()
			result = append(result, events...)
		}()
	}

	wg.Wait()

	slices.SortFunc(result, func(a, b Event) int {
		au := a.Date.Unix()
		bu := b.Date.Unix()
		if au < bu {
			return -1
		}
		if au > bu {
			return 1
		}
		return 0
	})

	err = nil
	if len(errList) > 0 {
		err = errors.Join(errList...)
	}

	return result, err
}

func parseCalendar(ctx context.Context, link string, tz *time.Location) ([]Event, error) {

	res, err := client.R().
		SetContext(ctx).
		Get(link)
	if err != nil {

		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {

		return nil, err
	}

	var events []Event
	doc.Find("div.fsCalendarDate").Each(func(_ int, div *goquery.Selection) {
		yearAttr := div.AttrOr("data-year", "")
		year, err := strconv.Atoi(yearAttr)
		if err != nil {

			return
		}

		monthAttr := div.AttrOr("data-month", "")
		month, err := strconv.Atoi(monthAttr)
		if err != nil {

			return
		}

		dayAttr := div.AttrOr("data-day", "")
		day, err := strconv.Atoi(dayAttr)
		if err != nil {

			return
		}

		div.Parent().Find("a.fsCalendarEventLink").Each(func(_ int, s *goquery.Selection) {
			name := s.Text()

			events = append(events, Event{
				Name: name,
				Date: time.Date(
					year, time.Month(month), day,
					0, 0, 0, 0, tz,
				),
			})
		})
	})

	return events, nil
}
