package vcs

import (
	"context"
	gradesnapshotpb "vcassist-backend/services/gradesnapshots/api"
	"vcassist-backend/services/studentdata/api"
)

func patchStudentDataWithGradeSnapshots(ctx context.Context, data *api.StudentData, gradesnapshotdata *gradesnapshotpb.PullResponse) {
	ctx, span := tracer.Start(ctx, "patchStudentData:WithGradeSnapshots")
	defer span.End()

	for i, snapshotcourse := range gradesnapshotdata.GetCourses() {
		var course *api.Course
		for _, c := range data.GetCourses() {
			if course.GetName() == c.GetName() {
				course = c
				break
			}
		}
		if course == nil {
			course = &api.Course{}
		}

		snapshots := make([]*api.GradeSnapshot, len(snapshotcourse.GetSnapshots()))
		for i, snap := range snapshotcourse.GetSnapshots() {
			snapshots[i] = &api.GradeSnapshot{
				Time:  snap.GetTime(),
				Value: snap.GetValue(),
			}
		}
		data.Courses[i] = &api.Course{
			Name:      snapshotcourse.GetCourse(),
			Snapshots: snapshots,
		}
	}
}
