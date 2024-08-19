package vcs

import (
	"context"
	gradesnapshotsv1 "vcassist-backend/proto/vcassist/services/gradesnapshots/v1"
	studentdatav1 "vcassist-backend/proto/vcassist/services/studentdata/v1"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func patchStudentDataWithGradeSnapshots(ctx context.Context, data *studentdatav1.StudentData, gradesnapshotdata *gradesnapshotsv1.PullResponse) {
	ctx, span := tracer.Start(ctx, "patch:gradesnapshots")
	defer span.End()

	unresolved := true

	for _, snapshotcourse := range gradesnapshotdata.GetCourses() {
		var course *studentdatav1.Course
		for _, pscourse := range data.GetCourses() {
			if snapshotcourse.GetCourse() == pscourse.GetName() {
				course = pscourse
				break
			}
		}
		if course == nil {
			continue
		}

		span.AddEvent(
			"resolved course",
			trace.WithAttributes(attribute.String("course", course.GetName())),
		)
		snapshots := make([]*studentdatav1.GradeSnapshot, len(snapshotcourse.GetSnapshots()))
		for i, snap := range snapshotcourse.GetSnapshots() {
			snapshots[i] = &studentdatav1.GradeSnapshot{
				Time:  snap.GetTime(),
				Value: snap.GetValue(),
			}
		}
		course.Snapshots = snapshots
		unresolved = false
	}

	if unresolved {
		span.SetStatus(codes.Error, "no gradesnapshots were linked to the given courses")
	}
}
