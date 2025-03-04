package powerschool

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	powerschoolv1 "vcassist-backend/api/vcassist/powerschool/v1"
	"vcassist-backend/internal/db"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	powerschool_base_url = "https://vcsnet.powerschool.com"

	report_db_query     = "db.query"
	report_pb_unmarshal = "pb.unmarshal"
	report_pb_marshal   = "pb.marshal"

	report_ps_get_email                    = "powerschool.get-email"
	report_ps_new_client                   = "powerschool.new-client"
	report_ps_login_oauth                  = "powerschool.login-oauth"
	report_ps_request                      = "powerschool.powerschool-request"
	report_ps_response_data                = "powerschool.powerschool-response-data"
	report_ps_postprocess                  = "powerschool.post-process"
	report_ps_coursemeeting_range          = "powerschool.coursemeeting-range"
	report_ps_disambiguated_student_course = "powerschool.disambiguated-student-course"

	report_snapshot_get_snapshots = "snapshot.get-snapshots"
	report_snapshot_make_snapshot = "snapshot.make-snapshot"
)

var whitespaceRegex = regexp.MustCompile(`\s+`)

func matchName(name string, matchers []string) bool {
	name = strings.ToLower(name)
	name = strings.Trim(name, " \n\t")
	name = whitespaceRegex.ReplaceAllString(name, "")
	for _, m := range matchers {
		if strings.Contains(name, m) {
			return true
		}
	}
	return false
}

func (p Powerschool) getCurrentWeek() (start time.Time, end time.Time) {
	now := p.time.Now()
	start = now.Add(-time.Hour * 24 * time.Duration(now.Weekday()))
	end = now.Add(time.Hour * 24 * time.Duration(time.Saturday-now.Weekday()))
	return start, end
}

func (p Powerschool) scrapeUser(ctx context.Context, acc db.PowerschoolAccount) error {
	client, err := newClient(powerschool_base_url, p.tel)
	if err != nil {
		p.tel.ReportBroken(report_ps_new_client, err, powerschool_base_url)
		return err
	}
	err = client.LoginOAuth(ctx, acc.AccessToken, acc.IDToken, acc.TokenType)
	if err != nil {
		p.tel.ReportBroken(report_ps_login_oauth, err, acc)
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
		p.tel.ReportBroken(report_ps_response_data, err, acc)
		return err
	}

	psStudent := allStudents.Profiles[0]
	studentData, err := client.GetStudentData(ctx, requestStudentData{
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
	start, stop := p.getCurrentWeek()

	p.tel.ReportDebug(
		report_ps_coursemeeting_range,
		fmt.Sprintf("start: %v", start),
		fmt.Sprintf("stop: %v", stop),
	)
	req := requestSchedule{
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

// appending this invisible unicode char to the end of a string indicates
// that it is a string with a distinction marker
const ps_distinction_marker = "​"

func (p Powerschool) applyDisambiguation(courseData []courseData) {
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
		p.tel.ReportDebug(report_ps_disambiguated_student_course, c.Name)
	}
}

// ScrapeUser implements its corresponding interface method.
func (p Powerschool) ScrapeUser(ctx context.Context, accountId int64) error {
	acc, err := p.db.GetPSAccountFromId(ctx, accountId)
	if err != nil {
		p.tel.ReportBroken(report_db_query, err, "GetPSAccountFromId", accountId)
		return err
	}
	return p.scrapeUser(ctx, acc)
}

// ScrapeAll runs scraping for all users' courses.
func (p Powerschool) ScrapeAll(ctx context.Context) error {
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
func (p Powerschool) QueryData(ctx context.Context, accountId int64) (*powerschoolv1.DataResponse, error) {
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

var psHomeworkPassKeywords = []string{
	"hwpass",
	"homeworkpass",
}
var psPeriodRegex = regexp.MustCompile(`(\d+)\((.+)\)`)

func (p Powerschool) toPbCourses(ctx context.Context, accountId int64, input []courseData) []*powerschoolv1.CourseData {
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

		now := p.time.Now().Unix()
		var overallGrade int64 = -1
		for _, term := range course.Terms {
			start, err := decodeTimestamp(term.Start)
			if err != nil {
				p.tel.ReportWarning(
					report_ps_postprocess,
					fmt.Errorf("failed to parse term start time: %w", err),
					term.Start,
				)
				continue
			}

			end, err := decodeTimestamp(term.End)
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
			if matchName(assign.Title, psHomeworkPassKeywords) && assign.PointsEarned != nil {
				homeworkPasses = int(*assign.PointsEarned)
				continue
			}

			dueDate, err := decodeTimestamp(assign.DueDate)
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

func (p Powerschool) patchPbCourseMeetings(out []*powerschoolv1.CourseData, input []courseMeeting) {
	if len(out) == 0 {
		return
	}

	for _, course := range out {
		for _, courseMeeting := range input {
			if course.GetGuid() != courseMeeting.CourseGuid {
				continue
			}

			start, err := decodeTimestamp(courseMeeting.Start)
			if err != nil {
				p.tel.ReportWarning(
					report_ps_postprocess,
					fmt.Errorf("failed to parse start date of CourseMeeting: %w", err),
					courseMeeting.Start,
				)
				continue
			}
			stop, err := decodeTimestamp(courseMeeting.Stop)
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

func (p Powerschool) toPbData(
	ctx context.Context,
	accountId int64,
	profile studentProfile,
	data *responseStudentData,
	courseMeetings []courseMeeting,
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
