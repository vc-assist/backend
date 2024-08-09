package vcs

import (
	"context"
	linkerv1 "vcassist-backend/proto/vcassist/services/linker/v1"
	"vcassist-backend/proto/vcassist/services/linker/v1/linkerv1connect"
	powerservicev1 "vcassist-backend/proto/vcassist/services/powerservice/v1"
	vcsmoodlev1 "vcassist-backend/proto/vcassist/services/vcsmoodle/v1"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
)

func linkMoodleToPowerschool(
	ctx context.Context,
	linker linkerv1connect.LinkerServiceClient,
	moodle *vcsmoodlev1.GetStudentDataResponse,
	ps *powerservicev1.GetStudentDataResponse,
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

	res, err := linker.Link(ctx, &connect.Request[linkerv1.LinkRequest]{
		Msg: &linkerv1.LinkRequest{
			Src: &linkerv1.Set{
				Name: "moodle",
				Keys: moodleKeys,
			},
			Dst: &linkerv1.Set{
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

func linkWeightsToPowerschool(
	ctx context.Context,
	linker linkerv1connect.LinkerServiceClient,
	ps *powerservicev1.GetStudentDataResponse,
) (weightData, error) {
	ctx, span := tracer.Start(ctx, "linkWeightsToPowerschool")
	defer span.End()

	courseNames := make([]string, len(ps.GetCourseData()))
	for i, c := range ps.GetCourseData() {
		courseNames[i] = c.GetName()
	}

	res, err := linker.Link(ctx, &connect.Request[linkerv1.LinkRequest]{
		Msg: &linkerv1.LinkRequest{
			Src: &linkerv1.Set{
				Name: "weights",
				Keys: weightCourseNames,
			},
			Dst: &linkerv1.Set{
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
