package psapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	powerschoolv1 "vcassist-backend/api/vcassist/powerschool/v1"
	"vcassist-backend/internal/db"
	"vcassist-backend/lib/scrapers/powerschool"
	"vcassist-backend/lib/textutil"
	"vcassist-backend/lib/timezone"

	"github.com/go-resty/resty/v2"
	"google.golang.org/protobuf/proto"
)

var homeworkPassesKeywords = []string{
	"hwpass",
	"homeworkpass",
}
var periodRegex = regexp.MustCompile(`(\d+)\((.+)\)`)

func (impl Implementation) toPbCourses(input []powerschool.CourseData) []*powerschoolv1.CourseData {
	courses := make([]*powerschoolv1.CourseData, len(input))
	for i, course := range input {
		currentDay := ""
		matches := periodRegex.FindStringSubmatch(course.Period)
		if len(matches) < 3 {
			impl.tel.ReportWarning(
				report_scraper_postprocess,
				fmt.Errorf("failed to parse period: not enough matches"),
				matches,
			)
		} else {
			currentDay = matches[2]
		}

		now := timezone.Now().Unix()
		var overallGrade int64 = -1
		for _, term := range course.Terms {
			start, err := powerschool.DecodeTimestamp(term.Start)
			if err != nil {
				impl.tel.ReportWarning(
					report_scraper_postprocess,
					fmt.Errorf("failed to parse term start time: %w", err),
					term.Start,
				)
				continue
			}

			end, err := powerschool.DecodeTimestamp(term.End)
			if err != nil {
				impl.tel.ReportWarning(
					report_scraper_postprocess,
					fmt.Errorf("failed to parse term end time: %w", err),
					term.End,
				)
				continue
			}

			if now >= start.Unix() && now < end.Unix() {
				overallGrade = int64(term.FinalGrade.Percent)
				break
			}
		}

		homeworkPasses := 0
		categories := make(map[string]struct{})
		var assignments []*powerschoolv1.AssignmentData
		for _, assign := range course.Assignments {
			if textutil.MatchName(assign.Title, homeworkPassesKeywords) && assign.PointsEarned != nil {
				homeworkPasses = int(*assign.PointsEarned)
				continue
			}

			dueDate, err := powerschool.DecodeTimestamp(assign.DueDate)
			if err != nil {
				impl.tel.ReportWarning(
					report_scraper_postprocess,
					fmt.Errorf("failed to parse assignment due date: %w", err),
					assign.DueDate,
				)
			}

			categories[assign.Category] = struct{}{}
			assignments = append(assignments, &powerschoolv1.AssignmentData{
				Title:          assign.Title,
				Category:       assign.Category,
				DueDate:        dueDate.Unix(),
				Description:    assign.Description,
				PointsEarned:   assign.PointsEarned,
				PointsPossible: assign.PointsPossible,
				IsMissing:      assign.AttributeMissing,
				IsLate:         assign.AttributeLate,
				IsCollected:    assign.AttributeCollected,
				IsExempt:       assign.AttributeExempt,
				IsIncomplete:   assign.AttributeIncomplete,
			})
		}

		courses[i] = &powerschoolv1.CourseData{
			Guid:         course.Guid,
			Name:         course.Name,
			Room:         course.Room,
			Period:       course.Period,
			Teacher:      fmt.Sprintf("%s %s", course.TeacherFirstName, course.TeacherLastName),
			TeacherEmail: course.TeacherEmail,
			Assignments:  assignments,
			Meetings:     nil,

			DayName:        currentDay,
			OverallGrade:   float32(overallGrade),
			HomeworkPasses: int32(homeworkPasses),
		}
	}

	return courses
}

func (impl Implementation) patchPbCourseMeetings(out []*powerschoolv1.CourseData, input []powerschool.CourseMeeting) {
	if len(out) == 0 {
		return
	}

	for _, course := range out {
		for _, courseMeeting := range input {
			if course.GetGuid() != courseMeeting.CourseGuid {
				continue
			}

			start, err := powerschool.DecodeTimestamp(courseMeeting.Start)
			if err != nil {
				impl.tel.ReportWarning(
					report_scraper_postprocess,
					fmt.Errorf("failed to parse start date of CourseMeeting: %w", err),
					courseMeeting.Start,
				)
				continue
			}
			stop, err := powerschool.DecodeTimestamp(courseMeeting.Stop)
			if err != nil {
				impl.tel.ReportWarning(
					report_scraper_postprocess,
					fmt.Errorf("failed to parse stop date of CourseMeeting: %w", err),
					courseMeeting.Stop,
				)
				continue
			}

			course.Meetings = append(course.Meetings, &powerschoolv1.Meeting{
				Start: start.Unix(),
				Stop:  stop.Unix(),
			})
		}
	}
}

// func toPbSchools(input []powerschool.SchoolData) []*powerschoolv1.SchoolData {
// 	schools := make([]*powerschoolv1.SchoolData, len(input))
// 	for i, school := range input {
// 		schools[i] = &powerschoolv1.SchoolData{
// 			Name:          school.Name,
// 			Fax:           school.Fax,
// 			Phone:         school.Phone,
// 			Email:         school.Email,
// 			StreetAddress: school.StreetAddress,
// 			City:          school.City,
// 			State:         school.State,
// 			Zip:           school.Zip,
// 			Country:       school.Country,
// 		}
// 	}
// 	return schools
// }

// func toPbBulletins(input []powerschool.Bulletin) []*powerschoolv1.Bulletin {
// 	bulletins := make([]*powerschoolv1.Bulletin, len(input))
// 	for i, bulletin := range input {
// 		start, err := powerschool.DecodeBulletinTimestamp(bulletin.StartDate)
// 		if err != nil {
// 			slog.Warn(
// 				"failed to parse bulletin start time",
// 				"time", bulletin.StartDate,
// 				"err", err,
// 			)
// 			continue
// 		}
// 		stop, err := powerschool.DecodeBulletinTimestamp(bulletin.EndDate)
// 		if err != nil {
// 			slog.Warn(
// 				"failed to parse bulletin end time",
// 				"time", bulletin.EndDate,
// 				"err", err,
// 			)
// 			continue
// 		}
//
// 		bulletins[i] = &powerschoolv1.Bulletin{
// 			Title:     bulletin.Title,
// 			Body:      bulletin.Body,
// 			StartDate: start.Unix(),
// 			EndDate:   stop.Unix(),
// 		}
// 	}
// 	return bulletins
// }

func (impl Implementation) toPbData(
	profile powerschool.StudentProfile,
	data *powerschool.GetStudentDataResponse,
	courseMeetings []powerschool.CourseMeeting,
) *powerschoolv1.DataResponse {
	gpa, err := strconv.ParseFloat(profile.CurrentGpa, 32)
	if err != nil {
		impl.tel.ReportBroken(
			report_scraper_postprocess,
			fmt.Errorf("parse gpa: %w", err),
			profile.CurrentGpa,
		)
	}

	if len(data.Student.Courses) == 0 {
		impl.tel.ReportBroken(
			report_scraper_postprocess,
			fmt.Errorf("student courses list is empty, only returning profile"),
		)

		return &powerschoolv1.DataResponse{
			Profile: &powerschoolv1.StudentProfile{
				CurrentGpa: float32(gpa),
				// the following fields are disabled for now as they don't yet have a use
				// Guid:       profile.Guid,
				// Name:       fmt.Sprintf("%s %s", profile.FirstName, profile.LastName),
				// Photo:      "",
			},
		}
	}

	courses := impl.toPbCourses(data.Student.Courses)
	impl.patchPbCourseMeetings(courses, courseMeetings)
	// schools := toSisSchools(profile.Schools)
	// bulletins := toSisBulletins(profile.Bulletins)

	return &powerschoolv1.DataResponse{
		Profile: &powerschoolv1.StudentProfile{
			// Guid:       profile.Guid,
			CurrentGpa: float32(gpa),
			// Name:       fmt.Sprintf("%s %s", profile.FirstName, profile.LastName),
		},
		// Schools:   schools,
		// Bulletins: bulletins,
		Courses: courses,
	}
}

// appending this invisible unicode char to the end of a string indicates
// that it is a string with a distinction marker
const distinctionMarker = "​"

func (impl Implementation) scrapeUser(ctx context.Context, client *powerschool.Client) (*powerschoolv1.DataResponse, error) {
	allStudents, err := client.GetAllStudents(ctx)
	if err != nil {
		return nil, err
	}
	if len(allStudents.Profiles) == 0 {
		return nil, fmt.Errorf(
			"could not find student profile, are your credentials expired?",
		)
	}

	psStudent := allStudents.Profiles[0]
	studentData, err := client.GetStudentData(ctx, powerschool.GetStudentDataRequest{
		Guid: psStudent.Guid,
	})
	if err != nil {
		return nil, err
	}

	for i, src := range studentData.Student.Courses {
		if strings.HasSuffix(src.Name, distinctionMarker) {
			continue
		}
		clarificationNeeded := false
		for j := i + 1; j < len(studentData.Student.Courses); j++ {
			dst := studentData.Student.Courses[j]
			if src.Name == dst.Name {
				clarificationNeeded = true
				dst.Name = fmt.Sprintf("%s %s"+distinctionMarker, dst.Name, dst.Period)
				// because this is not a pointer
				studentData.Student.Courses[j] = dst
			}
		}
		if clarificationNeeded {
			src.Name = fmt.Sprintf("%s %s"+distinctionMarker, src.Name, src.Period)
			// because this is not a pointer
			studentData.Student.Courses[i] = src
		}
	}

	for _, c := range studentData.Student.Courses {
		impl.tel.ReportDebug("student course", c.Name)
	}

	guids := make([]string, len(studentData.Student.Courses))
	for i, c := range studentData.Student.Courses {
		guids[i] = c.Guid
	}
	start, stop := timezone.GetCurrentWeek(timezone.Now())

	impl.tel.ReportDebug(
		"powerschool CourseMeeting range",
		fmt.Sprintf("start: %v", start),
		fmt.Sprintf("stop: %v", stop),
	)

	req := powerschool.GetCourseMeetingListRequest{
		CourseGuids: guids,
		Start:       start.Format(time.RFC3339),
		Stop:        stop.Format(time.RFC3339),
	}
	res, err := client.GetCourseMeetingList(ctx, req)
	if err != nil {
		impl.tel.ReportBroken(
			report_scraper_ps_request,
			fmt.Errorf("GetCourseMeetingList: %w", err),
			req,
		)
	}

	// MAY BE USED LATER, DO NOT DELETE
	// studentPhoto, err := client.GetStudentPhoto(ctx, scraper.GetStudentPhotoRequest{
	// 	Guid: psStudent.Guid,
	// })
	// if err != nil {
	// 	span.RecordError(err)
	// 	span.SetStatus(codes.Error, "failed to get student photo")
	// }

	return impl.toPbData(psStudent, studentData, res.Meetings), nil
}

// ScrapeAll runs scraping for all users' courses.
func (impl Implementation) ScrapeAll(ctx context.Context) error {
	accounts, err := impl.db.GetAllPSAccounts(ctx)
	if err != nil {
		impl.tel.ReportBroken(report_db_query, err, "GetAllPSAccounts")
		return err
	}
	wg := sync.WaitGroup{}
	for _, acc := range accounts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			impl.ScrapeUser(ctx, acc.ID)
		}()
	}
	return nil
}

const powerschool_base_url = "https://vcsnet.powerschool.com"

// ScrapeUser implements its corresponding interface method.
func (impl Implementation) ScrapeUser(ctx context.Context, accountId int64) error {
	acc, err := impl.db.GetPSAccountFromId(ctx, accountId)
	if err != nil {
		impl.tel.ReportBroken(report_db_query, err, "GetPSAccountFromEmail", accountId)
		return err
	}

	client, err := powerschool.NewClient(powerschool_base_url)
	if err != nil {
		impl.tel.ReportBroken(report_scraper_new_client, err, powerschool_base_url)
		return err
	}
	err = client.LoginOAuth(ctx, acc.Token)
	if err != nil {
		impl.tel.ReportBroken(report_scraper_login_oauth, err, acc.Token)
		return err
	}

	data, err := impl.scrapeUser(ctx, client)
	if err != nil {
		return err
	}

	buff, err := proto.Marshal(data)
	if err != nil {
		impl.tel.ReportBroken(report_pb_marshal, err)
		return err
	}

	err = impl.db.SetPSCachedData(ctx, db.SetPSCachedDataParams{
		AccountID: accountId,
		Data:      buff,
	})
	if err != nil {
		impl.tel.ReportBroken(report_db_query, err, "SetPSCachedData", accountId, len(buff))
		return err
	}

	return nil
}

// QueryData implements its corresponding interface method.
func (impl Implementation) QueryData(ctx context.Context, accountId int64) (*powerschoolv1.DataResponse, error) {
	cached, err := impl.db.GetPSCachedData(ctx, accountId)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no cached data found, please try again later")
	}
	if err != nil {
		impl.tel.ReportBroken(report_db_query, err, "GetPSCachedData")
		return nil, err
	}

	var res powerschoolv1.DataResponse
	err = proto.Unmarshal(cached, &res)
	if err != nil {
		impl.tel.ReportBroken(report_pb_unmarshal, err, len(cached))
		return nil, err
	}
	return &res, nil
}

var defaultClient = resty.New()

type googleUserInfo struct {
	Email string `json:"email"`
}

// GetEmail implements its corresponding interface method.
func (impl Implementation) GetEmail(ctx context.Context, token string) (email string, err error) {
	res, err := defaultClient.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", token)).
		Get("https://openidconnect.googleapis.com/v1/userinfo")
	if err != nil {
		impl.tel.ReportBroken(report_impl_get_email, err, "failed to make request", token)
		return "", err
	}
	if res.StatusCode() >= 400 || res.StatusCode() < 500 {
		impl.tel.ReportBroken(report_impl_get_email, err, "invalid token", token)
		return "", fmt.Errorf("invalid token")
	}
	body := res.Body()

	var result googleUserInfo
	err = json.Unmarshal(body, &result)
	if err != nil {
		impl.tel.ReportBroken(report_impl_get_email, err, "failed to unmarshal", string(body))
		return "", err
	}

	return result.Email, nil
}
