package vcsnet

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"time"
	"vcassist-backend/internal/chrono"
	"vcassist-backend/internal/telemetry"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
)

const (
	report_client_parse_calendar = "client.parse-calendar"
)

func GetYearRange(now time.Time) (int, int) {
	year := now.Year()
	month := now.Month()
	if month >= 8 && month <= 12 {
		return year, year + 1
	}
	return year - 1, year
}

type Client struct {
	http *resty.Client
	time chrono.TimeAPI
	tel  telemetry.API
}

func NewClient(time chrono.TimeAPI, tel telemetry.API) Client {
	return Client{
		http: resty.New(),
		time: time,
		tel:  tel,
	}
}

type SchoolYear struct {
	StartYear int
	EndYear   int
	StartTime time.Time
}

// GetSchoolYear gets the current school year, or if on summer break, the previous school year
func (c Client) GetSchoolYear() SchoolYear {
	now := c.time.Now()
	year := now.Year()
	month := now.Month()

	// encompasses S1
	if month >= 8 {
		return SchoolYear{
			StartYear: year,
			EndYear:   year + 1,
			StartTime: time.Date(year, 8, 1, 0, 0, 0, 0, chrono.LA()),
		}
	}

	// encompasses summer break & S2
	return SchoolYear{
		StartYear: year - 1,
		EndYear:   year,
		StartTime: time.Date(year-1, 8, 1, 0, 0, 0, 0, chrono.LA()),
	}
}

type Event struct {
	Name string
	Date time.Time
}

func (c Client) FetchEvents(ctx context.Context) ([]Event, error) {
	link, err := url.Parse("https://www.vcs.net/fs/elements/39337")
	if err != nil {
		return nil, err
	}

	schoolYear := c.GetSchoolYear()

	c.tel.ReportDebug("event bounds", schoolYear)

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

			events, err := c.parseCalendar(ctx, link.String())
			if err != nil {
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

func (c Client) parseCalendar(ctx context.Context, link string) ([]Event, error) {
	c.tel.ReportDebug("parse calendar", link)

	res, err := c.http.R().
		SetContext(ctx).
		Get(link)
	if err != nil {
		c.tel.ReportBroken(
			report_client_parse_calendar,
			fmt.Errorf("fetch: %w", err),
		)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
		c.tel.ReportBroken(
			report_client_parse_calendar,
			fmt.Errorf("parse html: %w", err),
		)
		return nil, err
	}

	var events []Event
	doc.Find("div.fsCalendarDate").Each(func(_ int, div *goquery.Selection) {
		yearAttr := div.AttrOr("data-year", "")
		year, err := strconv.Atoi(yearAttr)
		if err != nil {
			c.tel.ReportBroken(
				report_client_parse_calendar,
				fmt.Errorf("parse year: %w", err),
				yearAttr,
			)
			return
		}

		monthAttr := div.AttrOr("data-month", "")
		month, err := strconv.Atoi(monthAttr)
		if err != nil {
			c.tel.ReportBroken(
				report_client_parse_calendar,
				fmt.Errorf("parse month: %w", err),
				monthAttr,
			)
			return
		}

		dayAttr := div.AttrOr("data-day", "")
		day, err := strconv.Atoi(dayAttr)
		if err != nil {
			c.tel.ReportBroken(
				report_client_parse_calendar,
				fmt.Errorf("parse day: %w", err),
				dayAttr,
			)
			return
		}

		div.Parent().Find("a.fsCalendarEventLink").Each(func(_ int, s *goquery.Selection) {
			name := s.Text()

			c.tel.ReportDebug(
				"parsed event",
				fmt.Sprintf("%d/%d/%d", month, day, year),
				name,
			)

			events = append(events, Event{
				Name: name,
				Date: time.Date(
					year, time.Month(month), day,
					0, 0, 0, 0, chrono.LA(),
				),
			})
		})
	})

	return events, nil
}
