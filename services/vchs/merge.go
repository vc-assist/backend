package vchs

import (
	"context"
	"fmt"
	linkerpb "vcassist-backend/services/linker/api"
	"vcassist-backend/services/studentdata/api"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
)

func (s Service) mergeStudentData(
	ctx context.Context,
	psdata *api.StudentData,
	moodledata *api.StudentData,
) (*api.StudentData, error) {
	ctx, span := tracer.Start(ctx, "mergeStudentData")
	defer span.End()

	if psdata == nil {
		return moodledata, nil
	}
	if moodledata == nil {
		return psdata, nil
	}

	psCourseNames := make([]string, len(psdata.Courses))
	for i, c := range psdata.Courses {
		psCourseNames[i] = c.Name
	}
	moodleCourseNames := make([]string, len(moodledata.Courses))
	for i, c := range moodledata.Courses {
		moodleCourseNames[i] = c.Name
	}

	linkRes, err := s.linker.Link(ctx, &connect.Request[linkerpb.LinkRequest]{
		Msg: &linkerpb.LinkRequest{
			Src: &linkerpb.Set{
				Name: "powerschool",
				Keys: psCourseNames,
			},
			Dst: &linkerpb.Set{
				Name: "moodle",
				Keys: moodleCourseNames,
			},
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	for _, psCourse := range psdata.Courses {
		moodleName := linkRes.Msg.SrcToDst[psCourse.Name]
		var moodleCourse *api.Course
		for _, c := range moodledata.Courses {
			if c.Name == moodleName {
				moodleCourse = c
				break
			}
		}
		if moodleCourse == nil {
			err := fmt.Errorf("could not find moodle course by name '%s'", moodleName)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			continue
		}

		psCourse.RemoteMeetingLink = moodleCourse.RemoteMeetingLink
		psCourse.LessonPlan = moodleCourse.LessonPlan
	}

	return psdata, nil
}
