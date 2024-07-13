package vchs

import (
	"context"
	"vcassist-backend/services/studentdata/api"
	moodlepb "vcassist-backend/services/vchsmoodle/api"
)

func patchStudentDataWithMoodle(ctx context.Context, data *api.StudentData, moodledata *moodlepb.GetStudentDataResponse) {
	ctx, span := tracer.Start(ctx, "patchStudentData:WithMoodle")
	defer span.End()

	for _, moodleCourse := range moodledata.Courses {
		var course *api.Course
		for _, c := range data.Courses {
			if c.Name == moodleCourse.Name {
				course = c
				break
			}
		}
		if course == nil {
			course = &api.Course{}
		}

		course.RemoteMeetingLink = moodleCourse.ZoomLink
		course.LessonPlan = moodleCourse.LessonPlan
	}
}
