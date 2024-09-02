package vcsis

import (
	"context"
	"log/slog"
	linkerv1 "vcassist-backend/proto/vcassist/services/linker/v1"
	sisv1 "vcassist-backend/proto/vcassist/services/sis/v1"

	"connectrpc.com/connect"
)

// map[CourseName]map[CategoryName]<weight value: 0-1>
type WeightData = map[string]map[string]float32

func (s Service) addWeights(ctx context.Context, courseData []*sisv1.CourseData) error {
	courseNames := make([]string, len(courseData))
	for i, c := range courseData {
		courseNames[i] = c.GetName()
	}

	res, err := s.linker.Link(ctx, &connect.Request[linkerv1.LinkRequest]{
		Msg: &linkerv1.LinkRequest{
			Src: &linkerv1.Set{
				Name: "weights",
				Keys: s.weightCourseNames,
			},
			Dst: &linkerv1.Set{
				Name: "powerschool",
				Keys: courseNames,
			},
		},
	})
	if err != nil {
		return err
	}

	for weightCourseName, powerschoolName := range res.Msg.GetSrcToDst() {
		categories := s.weightData[weightCourseName]

		var target *sisv1.CourseData
		for _, course := range courseData {
			if course.GetName() == powerschoolName {
				target = course
				break
			}
		}
		if target == nil {
			slog.ErrorContext(
				ctx,
				"failed to find a powerschool course, this should never happen?",
				"weight_name", weightCourseName,
				"powerschool_name", powerschoolName,
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
	}
	return nil
}
