package vcs

import (
	"context"
	studentdatav1 "vcassist-backend/proto/vcassist/services/studentdata/v1"
)

func patchStudentDataWithWeights(ctx context.Context, data *studentdatav1.StudentData, weights weightData) {
	ctx, span := tracer.Start(ctx, "patchStudentDataWithWeights")
	defer span.End()

	for _, course := range data.GetCourses() {
		var weightList map[string]float32
		for courseName, w := range weights {
			if course.GetName() == courseName {
				weightList = w
				break
			}
		}
		if weightList == nil {
			continue
		}

		assignmentTypes := make([]*studentdatav1.AssignmentType, len(weightList))
		i := 0
		for name, value := range weightList {
			assignmentTypes[i] = &studentdatav1.AssignmentType{
				Name:   name,
				Weight: value,
			}
			i++
		}
		course.AssignmentTypes = assignmentTypes
	}
}
