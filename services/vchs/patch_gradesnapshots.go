package vchs

import (
	"context"
	gradesnapshotpb "vcassist-backend/services/gradesnapshots/api"
	"vcassist-backend/services/studentdata/api"
)

func patchStudentDataWithGradeSnapshots(ctx context.Context, data *api.StudentData, gradesnapshotdata *gradesnapshotpb.PullResponse) {
	ctx, span := tracer.Start(ctx, "patchStudentData:WithGradeSnapshots")
	defer span.End()

	for i, snapshotcourse := range gradesnapshotdata.Courses {
		var course *api.Course
		for _, c := range data.Courses {
			if course.Name == c.Name {
				course = c
				break
			}
		}
		if course == nil {
			course = &api.Course{}
		}

		snapshots := make([]*api.GradeSnapshot, len(snapshotcourse.Snapshots))
		for i, snap := range snapshotcourse.Snapshots {
			snapshots[i] = &api.GradeSnapshot{
				Time:  snap.Time,
				Value: snap.Value,
			}
		}
		data.Courses[i] = &api.Course{
			Name:      snapshotcourse.Course,
			Snapshots: snapshots,
		}
	}
}
