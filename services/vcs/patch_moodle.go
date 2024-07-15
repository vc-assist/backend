package vcs

import (
	"context"
	"vcassist-backend/services/studentdata/api"
	moodlepb "vcassist-backend/services/vcsmoodle/api"
)

func patchStudentDataWithMoodle(ctx context.Context, data *api.StudentData, moodledata *moodlepb.GetStudentDataResponse) {
	ctx, span := tracer.Start(ctx, "patchStudentData:WithMoodle")
	defer span.End()

	for _, moodleCourse := range moodledata.GetCourses() {
		var course *api.Course
		for _, c := range data.GetCourses() {
			if c.GetName() == moodleCourse.GetName() {
				course = c
				break
			}
		}
		if course == nil {
			course = &api.Course{}
		}

		course.RemoteMeetingLink = moodleCourse.GetZoomLink()
		course.LessonPlan = moodleCourse.GetLessonPlan()
	}
}
