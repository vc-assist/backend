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
	psData *api.StudentData,
	moodleData *api.StudentData,
	gradesnapshotData *api.StudentData,
) (*api.StudentData, error) {
	ctx, span := tracer.Start(ctx, "mergeStudentData")
	defer span.End()

	if psData == nil {
		return moodleData, nil
	}
	if moodleData == nil {
		return psData, nil
	}

	psCourseNames := make([]string, len(psData.Courses))
	for i, c := range psData.Courses {
		psCourseNames[i] = c.Name
	}
	moodleCourseNames := make([]string, len(moodleData.Courses))
	for i, c := range moodleData.Courses {
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

	for _, psCourse := range psData.Courses {
		moodleName := linkRes.Msg.SrcToDst[psCourse.Name]
		var moodleCourse *api.Course
		for _, c := range moodleData.Courses {
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

	for _, psCourse := range psData.Courses {
		var snapshotCourse *api.Course
		for _, c := range gradesnapshotData.Courses {
			if c.Name == psCourse.Name {
				snapshotCourse = c
				break
			}
		}
		if snapshotCourse == nil {
			err := fmt.Errorf("could not find snapshot course by name '%s'", psCourse.Name)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			continue
		}

		psCourse.Snapshots = snapshotCourse.Snapshots
	}

	return psData, nil
}
