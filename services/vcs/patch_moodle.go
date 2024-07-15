package vcs

import (
	"context"
	studentdatav1 "vcassist-backend/proto/vcassist/services/studentdata/v1"
	vcsmoodlev1 "vcassist-backend/proto/vcassist/services/vcsmoodle/v1"
)

func patchStudentDataWithMoodle(ctx context.Context, data *studentdatav1.StudentData, moodledata *vcsmoodlev1.GetStudentDataResponse) {
	ctx, span := tracer.Start(ctx, "patchStudentData:WithMoodle")
	defer span.End()

	for _, moodleCourse := range moodledata.GetCourses() {
		var course *studentdatav1.Course
		for _, c := range data.GetCourses() {
			if c.GetName() == moodleCourse.GetName() {
				course = c
				break
			}
		}
		if course == nil {
			course = &studentdatav1.Course{}
		}

		course.RemoteMeetingLink = moodleCourse.GetZoomLink()
		course.LessonPlan = moodleCourse.GetLessonPlan()
	}
}
