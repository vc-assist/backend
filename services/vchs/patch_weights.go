package vchs

import (
	"context"
	"vcassist-backend/services/studentdata/api"
)

func patchStudentDataWithWeights(ctx context.Context, data *api.StudentData, weights weightData) {
	ctx, span := tracer.Start(ctx, "patchStudentDataWithWeights")
	defer span.End()

	for _, course := range data.Courses {
		var weightList map[string]float32
		for courseName, w := range weights {
			if course.Name == courseName {
				weightList = w
				break
			}
		}
		if weightList == nil {
			continue
		}

		assignmentTypes := make([]*api.AssignmentType, len(weightList))
		i := 0
		for name, value := range weightList {
			assignmentTypes[i] = &api.AssignmentType{
				Name:   name,
				Weight: value,
			}
			i++
		}
		course.AssignmentTypes = assignmentTypes
	}
}
