package vcs

import (
	"context"
	linkerpb "vcassist-backend/services/linker/api"
	linkerrpc "vcassist-backend/services/linker/api/apiconnect"
	pspb "vcassist-backend/services/powerschool/api"
	moodlepb "vcassist-backend/services/vcsmoodle/api"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
)

func linkMoodleToPowerschool(
	ctx context.Context,
	linker linkerrpc.LinkerServiceClient,
	moodle *moodlepb.GetStudentDataResponse,
	ps *pspb.GetStudentDataResponse,
) error {
	ctx, span := tracer.Start(ctx, "linkMoodleToPowerschool")
	defer span.End()

	moodleKeys := make([]string, len(moodle.GetCourses()))
	for i, c := range moodle.GetCourses() {
		moodleKeys[i] = c.GetName()
	}
	psKeys := make([]string, len(ps.GetCourseData()))
	for i, c := range ps.GetCourseData() {
		psKeys[i] = c.GetName()
	}

	res, err := linker.Link(ctx, &connect.Request[linkerpb.LinkRequest]{
		Msg: &linkerpb.LinkRequest{
			Src: &linkerpb.Set{
				Name: "moodle",
				Keys: moodleKeys,
			},
			Dst: &linkerpb.Set{
				Name: "powerschool",
				Keys: psKeys,
			},
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	for _, c := range moodle.GetCourses() {
		c.Name = res.Msg.GetSrcToDst()[c.GetName()]
	}
	return nil
}

func getWeightsForPowerschool(
	ctx context.Context,
	linker linkerrpc.LinkerServiceClient,
	ps *pspb.GetStudentDataResponse,
) (weightData, error) {
	ctx, span := tracer.Start(ctx, "getWeightsForPowerschool")
	defer span.End()

	courseNames := make([]string, len(ps.GetCourseData()))
	for i, c := range ps.GetCourseData() {
		courseNames[i] = c.GetName()
	}

	res, err := linker.Link(ctx, &connect.Request[linkerpb.LinkRequest]{
		Msg: &linkerpb.LinkRequest{
			Src: &linkerpb.Set{
				Name: "weights",
				Keys: weightCourseNames,
			},
			Dst: &linkerpb.Set{
				Name: "powerschool",
				Keys: courseNames,
			},
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	data := make(weightData)
	for weightCourseName, powerschoolName := range res.Msg.GetSrcToDst() {
		data[powerschoolName] = weights[weightCourseName]
	}
	return data, nil
}
