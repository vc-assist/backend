package vcsis

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"vcassist-backend/lib/gradestore"
	"vcassist-backend/lib/scrapers/powerschool"
	scraper "vcassist-backend/lib/scrapers/powerschool"
	"vcassist-backend/lib/timezone"
	sisv1 "vcassist-backend/proto/vcassist/services/sis/v1"

	_ "embed"

	"github.com/antzucaro/matchr"
)

// appending this invisible unicode char to the end of a string indicates
// that it is a string with a distinction marker
const distinctionMarker = "​"

func ScrapePowerschool(ctx context.Context, client *powerschool.Client) (*sisv1.Data, error) {
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
	studentData, err := client.GetStudentData(ctx, scraper.GetStudentDataRequest{
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
		slog.Debug("student course", "name", c.Name)
	}

	guids := make([]string, len(studentData.Student.Courses))
	for i, c := range studentData.Student.Courses {
		guids[i] = c.Guid
	}
	start, stop := timezone.GetCurrentWeek(timezone.Now())

	slog.Debug("powerschool CourseMeeting range", "start", start, "stop", stop)
	res, err := client.GetCourseMeetingList(ctx, scraper.GetCourseMeetingListRequest{
		CourseGuids: guids,
		Start:       start.Format(time.RFC3339),
		Stop:        stop.Format(time.RFC3339),
	})
	if err != nil {
		slog.WarnContext(
			ctx,
			"fetch course meetings",
			"err", err,
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

	return powerschool.ToSISData(ctx, psStudent, studentData, res.Meetings), nil
}

func AddGradeSnapshots(ctx context.Context, courseData []*sisv1.CourseData, series []gradestore.CourseSnapshotSeries) {
	for _, course := range series {
		var target *sisv1.CourseData
		for _, tc := range courseData {
			if tc.GetGuid() == course.Course {
				target = tc
				break
			}
		}
		if target == nil {
			slog.WarnContext(ctx, "failed to find saved grade snapshot course in powerschool data", "course", course.Course)
			continue
		}

		snapshots := make([]*sisv1.GradeSnapshot, len(course.Snapshots))
		for i, s := range course.Snapshots {
			snapshots[i] = &sisv1.GradeSnapshot{
				Time:  s.Time.Unix(),
				Value: s.Value,
			}
		}
		target.Snapshots = snapshots
	}
}

// map[CourseName]map[CategoryName]<weight value: 0-1>
type WeightData = map[string]map[string]float32

func AddWeights(
	ctx context.Context,
	courseData []*sisv1.CourseData,
	weightData WeightData,
	powerschoolToWeightsMap map[string]string,
) {
	for powerschoolName, weightName := range powerschoolToWeightsMap {
		categories := weightData[weightName]

		var target *sisv1.CourseData
		for _, course := range courseData {
			if course.GetName() == powerschoolName {
				target = course
				break
			}
		}
		if target == nil {
			var psNames []string
			for _, c := range courseData {
				psNames = append(psNames, c.GetName())
			}
			slog.ErrorContext(
				ctx,
				"failed to find powerschool course name had been provided to linker, this should never happen!",
				"weight_name", weightName,
				"powerschool_name", powerschoolName,
				"powerschool_name_list", psNames,
			)
			continue
		}

		out := make([]*sisv1.AssignmentCategory, len(categories))
		i := 0
		for category, weight := range categories {
			out[i] = &sisv1.AssignmentCategory{
				Name:   category,
				Weight: weight,
			}
			i++
		}
		target.AssignmentCategories = out

		for _, a := range target.Assignments {
			_, ok := categories[a.GetCategory()]
			if ok {
				continue
			}

			mostSimilar := ""
			var similarity float64
			for target := range categories {
				sim := matchr.JaroWinkler(a.GetCategory(), target, false)
				if sim > similarity {
					similarity = sim
					mostSimilar = target
				}
			}
			if mostSimilar != "" {
				a.Category = mostSimilar
			}
		}
	}
}
