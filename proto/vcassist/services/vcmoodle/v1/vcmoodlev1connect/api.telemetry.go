package vcmoodlev1connect

import (
	"context"

	connect "connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
	v1 "vcassist-backend/proto/vcassist/services/vcmoodle/v1"
)

type TracerLike interface {
	Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
}

var (
	MoodleServiceTracer TracerLike = otel.Tracer("vcassist.services.vcmoodle.v1.MoodleService")
)

type InstrumentedMoodleServiceClient struct {
	inner           MoodleServiceClient
	WithInputOutput bool
}

func NewInstrumentedMoodleServiceClient(inner MoodleServiceClient) InstrumentedMoodleServiceClient {
	return InstrumentedMoodleServiceClient{inner: inner}
}

func (c InstrumentedMoodleServiceClient) GetAuthStatus(ctx context.Context, req *connect.Request[v1.GetAuthStatusRequest]) (*connect.Response[v1.GetAuthStatusResponse], error) {

	res, err := c.inner.GetAuthStatus(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (c InstrumentedMoodleServiceClient) ProvideUsernamePassword(ctx context.Context, req *connect.Request[v1.ProvideUsernamePasswordRequest]) (*connect.Response[v1.ProvideUsernamePasswordResponse], error) {

	res, err := c.inner.ProvideUsernamePassword(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (c InstrumentedMoodleServiceClient) GetSession(ctx context.Context, req *connect.Request[v1.GetSessionRequest]) (*connect.Response[v1.GetSessionResponse], error) {

	res, err := c.inner.GetSession(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (c InstrumentedMoodleServiceClient) GetCourses(ctx context.Context, req *connect.Request[v1.GetCoursesRequest]) (*connect.Response[v1.GetCoursesResponse], error) {

	res, err := c.inner.GetCourses(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (c InstrumentedMoodleServiceClient) RefreshCourses(ctx context.Context, req *connect.Request[v1.RefreshCoursesRequest]) (*connect.Response[v1.RefreshCoursesResponse], error) {

	res, err := c.inner.RefreshCourses(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (c InstrumentedMoodleServiceClient) GetChapterContent(ctx context.Context, req *connect.Request[v1.GetChapterContentRequest]) (*connect.Response[v1.GetChapterContentResponse], error) {

	res, err := c.inner.GetChapterContent(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (c InstrumentedMoodleServiceClient) GetFileContent(ctx context.Context, req *connect.Request[v1.GetFileContentRequest]) (*connect.Response[v1.GetFileContentResponse], error) {

	res, err := c.inner.GetFileContent(ctx, req)
	if err != nil {

		return nil, err
	}

	return res, nil
}
