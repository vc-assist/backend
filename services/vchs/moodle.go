package vchs

import (
	"context"
	"vcassist-backend/services/studentdata/api"
	moodlepb "vcassist-backend/services/vchsmoodle/api"

	"connectrpc.com/connect"
)

func (s Service) studentDataFromMoodle(ctx context.Context, userEmail string) (*api.StudentData, error) {
	res, err := s.moodle.GetStudentData(ctx, &connect.Request[moodlepb.GetStudentDataRequest]{
		Msg: &moodlepb.GetStudentDataRequest{
			StudentId: userEmail,
		},
	})
	if err != nil {
		return nil, err
	}
	courseList := make([]*api.Course, len(res.Msg.Courses))
	for i, course := range res.Msg.Courses {
		courseList[i] = &api.Course{
			Name:              course.Name,
			RemoteMeetingLink: course.ZoomLink,
			LessonPlan:        course.LessonPlan,
		}
	}
	return &api.StudentData{Courses: courseList}, nil
}
