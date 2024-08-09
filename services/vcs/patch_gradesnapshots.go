package vcs

import (
	"context"
	gradesnapshotsv1 "vcassist-backend/proto/vcassist/services/gradesnapshots/v1"
	studentdatav1 "vcassist-backend/proto/vcassist/services/studentdata/v1"
)

func patchStudentDataWithGradeSnapshots(ctx context.Context, data *studentdatav1.StudentData, gradesnapshotdata *gradesnapshotsv1.PullResponse) {
	ctx, span := tracer.Start(ctx, "patchStudentData:WithGradeSnapshots")
	defer span.End()

	for _, snapshotcourse := range gradesnapshotdata.GetCourses() {
		var course *studentdatav1.Course
		var courseIdx int
		for i, c := range data.GetCourses() {
			if course.GetName() == c.GetName() {
				course = c
				courseIdx = i
				break
			}
		}
		if course == nil {
			continue
		}

		snapshots := make([]*studentdatav1.GradeSnapshot, len(snapshotcourse.GetSnapshots()))
		for i, snap := range snapshotcourse.GetSnapshots() {
			snapshots[i] = &studentdatav1.GradeSnapshot{
				Time:  snap.GetTime(),
				Value: snap.GetValue(),
			}
		}
		data.Courses[courseIdx] = &studentdatav1.Course{
			Name:      snapshotcourse.GetCourse(),
			Snapshots: snapshots,
		}
	}
}
