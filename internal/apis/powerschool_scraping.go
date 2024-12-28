package apis

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	powerschoolv1 "vcassist-backend/api/vcassist/powerschool/v1"
	"vcassist-backend/internal/db"
	"vcassist-backend/lib/scrapers/powerschool"
	"vcassist-backend/lib/textutil"
	"vcassist-backend/lib/timezone"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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

var psHomeworkPassKeywords = []string{
	"hwpass",
	"homeworkpass",
}
var psPeriodRegex = regexp.MustCompile(`(\d+)\((.+)\)`)

func (p PowerschoolImpl) toPbCourses(ctx context.Context, accountId int64, input []powerschool.CourseData) []*powerschoolv1.CourseData {
	courses := make([]*powerschoolv1.CourseData, len(input))
	for i, course := range input {
		currentDay := ""
		matches := psPeriodRegex.FindStringSubmatch(course.Period)
		if len(matches) < 3 {
			p.tel.ReportWarning(
				report_ps_postprocess,
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
				p.tel.ReportWarning(
					report_ps_postprocess,
					fmt.Errorf("failed to parse term start time: %w", err),
					term.Start,
				)
				continue
			}

			end, err := powerschool.DecodeTimestamp(term.End)
			if err != nil {
				p.tel.ReportWarning(
					report_ps_postprocess,
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
		var categories []string
		var assignments []*powerschoolv1.AssignmentData
		for _, assign := range course.Assignments {
			if textutil.MatchName(assign.Title, psHomeworkPassKeywords) && assign.PointsEarned != nil {
				homeworkPasses = int(*assign.PointsEarned)
				continue
			}

			dueDate, err := powerschool.DecodeTimestamp(assign.DueDate)
			if err != nil {
				p.tel.ReportWarning(
					report_ps_postprocess,
					fmt.Errorf("failed to parse assignment due date: %w", err),
					assign.DueDate,
				)
			}

			if !slices.Contains(categories, assign.Category) {
				categories = append(categories, assign.Category)
			}

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

		assignmentCategories := make([]*powerschoolv1.AssignmentCategory, len(categories))
		weightValues, err := p.weights.GetWeights(ctx, course.Name, categories)
		if err == nil {
			for i := range assignmentCategories {
				assignmentCategories[i] = &powerschoolv1.AssignmentCategory{
					Name:   categories[i],
					Weight: weightValues[i],
				}
			}
		}

		err = p.snapshot.MakeSnapshot(ctx, accountId, course.Guid, float32(overallGrade))
		if err != nil {
			p.tel.ReportBroken(
				report_snapshot_make_snapshot,
				err,
				accountId,
				course.Guid,
				float32(overallGrade),
			)
		}

		snapshots, err := p.snapshot.GetSnapshots(ctx, accountId, course.Guid)
		if err != nil {
			p.tel.ReportBroken(report_snapshot_get_snapshots, err, accountId, course.Guid)
		}
		pbSnapshots := make([]*powerschoolv1.GradeSnapshot, len(snapshots))
		for i, s := range snapshots {
			pbSnapshots[i] = &powerschoolv1.GradeSnapshot{
				Time:  timestamppb.New(s.Time),
				Value: s.Value,
			}
		}

		courses[i] = &powerschoolv1.CourseData{
			Guid:                 course.Guid,
			Name:                 course.Name,
			Room:                 course.Room,
			Period:               course.Period,
			Teacher:              fmt.Sprintf("%s %s", course.TeacherFirstName, course.TeacherLastName),
			TeacherEmail:         course.TeacherEmail,
			Assignments:          assignments,
			Meetings:             nil,
			AssignmentCategories: assignmentCategories,
			Snapshots:            pbSnapshots,

			DayName:        currentDay,
			OverallGrade:   float32(overallGrade),
			HomeworkPasses: int32(homeworkPasses),
		}
	}

	return courses
}

func (p PowerschoolImpl) patchPbCourseMeetings(out []*powerschoolv1.CourseData, input []powerschool.CourseMeeting) {
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
				p.tel.ReportWarning(
					report_ps_postprocess,
					fmt.Errorf("failed to parse start date of CourseMeeting: %w", err),
					courseMeeting.Start,
				)
				continue
			}
			stop, err := powerschool.DecodeTimestamp(courseMeeting.Stop)
			if err != nil {
				p.tel.ReportWarning(
					report_ps_postprocess,
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

func (p PowerschoolImpl) toPbData(
	ctx context.Context,
	accountId int64,
	profile powerschool.StudentProfile,
	data *powerschool.GetStudentDataResponse,
	courseMeetings []powerschool.CourseMeeting,
) *powerschoolv1.DataResponse {
	gpa, err := strconv.ParseFloat(profile.CurrentGpa, 32)
	if err != nil {
		p.tel.ReportBroken(
			report_ps_postprocess,
			fmt.Errorf("parse gpa: %w", err),
			profile.CurrentGpa,
		)
	}

	if len(data.Student.Courses) == 0 {
		p.tel.ReportBroken(
			report_ps_postprocess,
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

	courses := p.toPbCourses(ctx, accountId, data.Student.Courses)
	p.patchPbCourseMeetings(courses, courseMeetings)
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
const ps_distinction_marker = "​"

func (p PowerschoolImpl) applyDisambiguation(courseData []powerschool.CourseData) {
	for i, src := range courseData {
		if strings.HasSuffix(src.Name, ps_distinction_marker) {
			continue
		}
		clarificationNeeded := false
		for j := i + 1; j < len(courseData); j++ {
			dst := courseData[j]
			if src.Name == dst.Name {
				clarificationNeeded = true
				dst.Name = fmt.Sprintf("%s %s"+ps_distinction_marker, dst.Name, dst.Period)
				// because this is not a pointer
				courseData[j] = dst
			}
		}
		if clarificationNeeded {
			src.Name = fmt.Sprintf("%s %s"+ps_distinction_marker, src.Name, src.Period)
			// because this is not a pointer
			courseData[i] = src
		}
	}

	for _, c := range courseData {
		p.tel.ReportDebug("disambiguated student course", c.Name)
	}
}

const powerschool_base_url = "https://vcsnet.powerschool.com"

func (p PowerschoolImpl) createPSClient(ctx context.Context, token string) (*powerschool.Client, error) {
	client, err := powerschool.NewClient(powerschool_base_url)
	if err != nil {
		p.tel.ReportBroken(report_ps_new_client, err, powerschool_base_url)
		return nil, err
	}
	err = client.LoginOAuth(ctx, token)
	if err != nil {
		p.tel.ReportBroken(report_ps_login_oauth, err, token)
		return nil, err
	}
	return client, nil
}

func (p PowerschoolImpl) scrapeUser(ctx context.Context, acc db.PowerschoolAccount) error {
	client, err := p.createPSClient(ctx, acc.Token)
	if err != nil {
		return err
	}

	allStudents, err := client.GetAllStudents(ctx)
	if err != nil {
		err = fmt.Errorf("GetAllStudents: %w", err)
		p.tel.ReportBroken(report_ps_request, err)
		return err
	}
	if len(allStudents.Profiles) == 0 {
		err = fmt.Errorf("GetAllStudents: could not find student profile (credentials may be expired)")
		p.tel.ReportBroken(report_ps_response_data, err, acc.Token)
		return err
	}

	psStudent := allStudents.Profiles[0]
	studentData, err := client.GetStudentData(ctx, powerschool.GetStudentDataRequest{
		Guid: psStudent.Guid,
	})
	if err != nil {
		err = fmt.Errorf("GetStudentData: %w", err)
		p.tel.ReportBroken(report_ps_request, err, psStudent.Guid)
		return err
	}

	p.applyDisambiguation(studentData.Student.Courses)

	guids := make([]string, len(studentData.Student.Courses))
	for i, c := range studentData.Student.Courses {
		guids[i] = c.Guid
	}
	start, stop := timezone.GetCurrentWeek(timezone.Now())

	p.tel.ReportDebug(
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
		p.tel.ReportBroken(
			report_ps_request,
			fmt.Errorf("GetCourseMeetingList: %w", err),
			req,
		)
	}

	// currently unused
	// studentPhoto, err := client.GetStudentPhoto(ctx, scraper.GetStudentPhotoRequest{
	// 	Guid: psStudent.Guid,
	// })
	// if err != nil {
	// 	span.RecordError(err)
	// 	span.SetStatus(codes.Error, "failed to get student photo")
	// }

	data := p.toPbData(ctx, acc.ID, psStudent, studentData, res.Meetings)

	buff, err := proto.Marshal(data)
	if err != nil {
		p.tel.ReportBroken(report_pb_marshal, err)
		return err
	}

	err = p.db.AddPSCachedData(ctx, db.AddPSCachedDataParams{
		AccountID: acc.ID,
		Data:      buff,
	})
	if err != nil {
		p.tel.ReportBroken(report_db_query, err, "AddPSCachedData", acc.ID, len(buff))
		return err
	}

	return nil
}

// ScrapeUser implements its corresponding interface method.
func (p PowerschoolImpl) ScrapeUser(ctx context.Context, accountId int64) error {
	acc, err := p.db.GetPSAccountFromId(ctx, accountId)
	if err != nil {
		p.tel.ReportBroken(report_db_query, err, "GetPSAccountFromId", accountId)
		return err
	}
	return p.scrapeUser(ctx, acc)
}

// ScrapeAll runs scraping for all users' courses.
func (p PowerschoolImpl) ScrapeAll(ctx context.Context) error {
	accounts, err := p.db.GetAllPSAccounts(ctx)
	if err != nil {
		p.tel.ReportBroken(report_db_query, err, "GetAllPSAccounts")
		return err
	}
	wg := sync.WaitGroup{}
	for _, acc := range accounts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.scrapeUser(ctx, acc)
		}()
	}
	return nil
}

// QueryData implements its corresponding interface method.
func (p PowerschoolImpl) QueryData(ctx context.Context, accountId int64) (*powerschoolv1.DataResponse, error) {
	cached, err := p.db.GetPSCachedData(ctx, accountId)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no cached data found, please try again later")
	}
	if err != nil {
		p.tel.ReportBroken(report_db_query, err, "GetPSCachedData")
		return nil, err
	}

	var res powerschoolv1.DataResponse
	err = proto.Unmarshal(cached, &res)
	if err != nil {
		p.tel.ReportBroken(report_pb_unmarshal, err, len(cached))
		return nil, err
	}
	return &res, nil
}

type googleUserInfo struct {
	Email string `json:"email"`
}

// GetEmail implements its corresponding interface method.
func (p PowerschoolImpl) GetEmail(ctx context.Context, token string) (email string, err error) {
	res, err := defaultClient.R().
		SetContext(ctx).
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", token)).
		Get("https://openidconnect.googleapis.com/v1/userinfo")
	if err != nil {
		p.tel.ReportBroken(report_ps_get_email, err, "failed to make request", token)
		return "", err
	}
	if res.StatusCode() >= 400 || res.StatusCode() < 500 {
		p.tel.ReportBroken(report_ps_get_email, err, "invalid token", token)
		return "", fmt.Errorf("invalid token")
	}
	body := res.Body()

	var result googleUserInfo
	err = json.Unmarshal(body, &result)
	if err != nil {
		p.tel.ReportBroken(report_ps_get_email, err, "failed to unmarshal", string(body))
		return "", err
	}

	return result.Email, nil
}
