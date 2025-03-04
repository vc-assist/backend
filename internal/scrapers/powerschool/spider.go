package powerschool

import (
	"encoding/json"
	"fmt"
	"time"

	"vcassist-backend/internal/chrono"

	"github.com/LQR471814/scavenge"
	"github.com/LQR471814/scavenge/downloader"
)

type Session struct {
	AccessToken string
	IdToken     string
	TokenType   string
}

func (c Session) request(baseUrl, fragment string) *downloader.Request {
	req := downloader.POSTRequest(downloader.MustParseUrl(
		"https://mobile.powerschool.com/v3.0/graphql#" + fragment,
	))
	req.Headers.Add("Authorization", fmt.Sprintf("%s %s", c.TokenType, c.AccessToken))
	req.Headers.Add("profileUri", fmt.Sprintf("%s %s", c.IdToken))
	req.Headers.Add("ServerURL", baseUrl)
	return req
}

type Spider struct {
	serverUrl string
	users     []Session
	time      chrono.TimeAPI
}

func NewPSSpider(
	serverUrl string,
	users []Session,
	time chrono.TimeAPI,
) Spider {
	return Spider{
		serverUrl: serverUrl,
		users:     users,
		time:      time,
	}
}

const (
	request_all_students  = "AllStudentsFirstLevel"
	request_schedule      = "SectionMeetings"
	request_student_data  = "AllStudentData"
	request_student_photo = "StudentPhoto"
)

type requestMeta struct {
	user   Session
	method string
}

func (s Spider) makeRequest(
	user Session,
	name string,
	query string,
	variable any,
) *downloader.Request {
	req := user.request(s.serverUrl, request_all_students)
	body, err := json.Marshal(graphqlRequest{
		Name:     request_all_students,
		Query:    graphql_all_students,
		Variable: struct{}{},
	})
	if err != nil {
		// there is virtually nothing that would cause this to error unless something
		// has fundamentally gone wrong so it is correct to panic here
		panic(err)
	}
	req.SetBodyJSON(string(body))
	req.AddMeta(requestMeta{
		method: request_all_students,
	})
	return req
}

func (s Spider) StartingRequests() []*downloader.Request {
	requests := make([]*downloader.Request, len(s.users))

	for i, u := range s.users {
		req := s.makeRequest(u, request_all_students, graphql_all_students, struct{}{})
		requests[i] = req
	}

	return requests
}

func (s Spider) getCurrentWeek() (start time.Time, end time.Time) {
	now := s.time.Now()
	start = now.Add(-time.Hour * 24 * time.Duration(now.Weekday()))
	end = now.Add(time.Hour * 24 * time.Duration(time.Saturday-now.Weekday()))
	return start, end
}

func (s Spider) HandleResponse(nav scavenge.Navigator, res *downloader.Response) error {
	r := res.Request()
	meta, ok := downloader.GetRequestMeta[requestMeta](r)
	if !ok {
		panic("got request without requestMeta")
	}

	switch meta.method {
	case request_all_students:
		var out responseAllStudents
		err := res.JsonBody(&out)
		if err != nil {
			return err
		}
		if len(out.Profiles) == 0 {
			return fmt.Errorf("no student profiles found")
		}
		psStudent := out.Profiles[0]

		nav.Request(s.makeRequest(
			meta.user,
			request_student_data,
			graphql_student_data,
			requestStudentData{
				Guid: psStudent.Guid,
			},
		))
		return nil
	case request_student_data:
		var out responseStudentData
		err := res.JsonBody(&out)
		if err != nil {
			return err
		}

		guids := make([]string, len(out.Student.Courses))
		for i, c := range out.Student.Courses {
			guids[i] = c.Guid
		}
		start, stop := s.getCurrentWeek()

		nav.Request(s.makeRequest(
			meta.user,
			request_schedule,
			query_schedule,
			requestSchedule{
				CourseGuids: guids,
				Start:       start.Format(time.RFC3339),
				Stop:        stop.Format(time.RFC3339),
			},
		))

	}

}
