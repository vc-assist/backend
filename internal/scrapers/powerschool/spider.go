package powerschool

import (
	"encoding/json"
	"fmt"
	"time"
	"vcassist-backend/internal/chrono"

	"github.com/LQR471814/scavenge"
	"github.com/LQR471814/scavenge/downloader"
)

type PSCredentials struct {
	AccessToken string
	IdToken     string
	TokenType   string
}

func (c PSCredentials) baseRequest(baseUrl, fragment string) *downloader.Request {
	req := downloader.POSTRequest(downloader.MustParseUrl(
		"https://mobile.powerschool.com/v3.0/graphql#" + fragment,
	))
	req.Headers.Add("Authorization", fmt.Sprintf("%s %s", c.TokenType, c.AccessToken))
	req.Headers.Add("profileUri", fmt.Sprintf("%s %s", c.IdToken))
	req.Headers.Add("ServerURL", baseUrl)
	return req
}

type PSSpider struct {
	serverUrl string
	users     []PSCredentials
	time      chrono.TimeAPI
}

func NewPSSpider(
	serverUrl string,
	users []PSCredentials,
	time chrono.TimeAPI,
) PSSpider {
	return PSSpider{
		serverUrl: serverUrl,
		users:     users,
		time:      time,
	}
}

const (
	request_all_students     = "AllStudentsFirstLevel"
	request_section_meetings = "SectionMeetings"
)

func (s PSSpider) getCurrentWeek() (start, end time.Time) {
	now := s.time.Now()
	start = now.Add(-time.Hour * 24 * time.Duration(now.Weekday()))
	end = now.Add(time.Hour * 24 * time.Duration(time.Saturday-now.Weekday()))
	return start, end
}

func (s PSSpider) StartingRequests() []*downloader.Request {
	requests := make([]*downloader.Request, 0, 3*len(s.users))

	for _, u := range s.users {
		req := u.baseRequest(s.serverUrl, request_all_students)
		body, err := json.Marshal(graphqlRequest{
			Name:     request_all_students,
			Query:    query_all_students,
			Variable: struct{}{},
		})
		if err != nil {
			// there is virtually nothing that would cause this to error unless something
			// has fundamentally gone wrong so it is correct to panic here
			panic(err)
		}
		req.SetBodyJSON(string(body))
		requests = append(requests, req)

		req = u.baseRequest(s.serverUrl, request_section_meetings)
		start, end := s.getCurrentWeek()
		body, err = json.Marshal(graphqlRequest{
			Name:  request_section_meetings,
			Query: query_schedule,
			Variable: getCourseMeetingListRequest{
				CourseGuids: []string{}, // TODO: add guids
				Start:       start.Format(time.RFC3339),
				Stop:        end.Format(time.RFC3339),
			},
		})
		if err != nil {
			panic(err)
		}
		req.SetBodyJSON(string(body))
		requests = append(requests, req)
	}

	return requests
}

func (s PSSpider) HandleResponse(nav scavenge.Navigator, res *downloader.Response) error {
	switch res.Request().Url.Fragment {
	case request_all_students:
		nav.Request()
	}

}
