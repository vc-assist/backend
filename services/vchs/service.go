package vchs

import (
	"context"
	"errors"
	pspb "vcassist-backend/services/powerschool/api"
	psrpc "vcassist-backend/services/powerschool/api/apiconnect"
	"vcassist-backend/services/vchs/api"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("services/vchs")

type Service struct {
	powerschool psrpc.PowerschoolServiceClient
}

func (s Service) GetCredentialStatus(ctx context.Context, req *connect.Request[api.GetCredentialStatusRequest]) (*connect.Response[api.GetCredentialStatusResponse], error) {
	s.powerschool.GetAuthStatus(ctx, &connect.Request[pspb.GetAuthStatusRequest]{
		Msg: &pspb.GetAuthStatusRequest{},
	})

	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("services.vchs.api.VchsService.GetCredentialStatus is not implemented"))
}

func (s Service) ProvidePowerschoolToken(ctx context.Context, req *connect.Request[api.ProvidePowerschoolTokenRequest]) (*connect.Response[api.ProvidePowerschoolTokenResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("services.vchs.api.VchsService.ProvidePowerschoolToken is not implemented"))
}

func (s Service) ProvideMoodleLogin(ctx context.Context, req *connect.Request[api.ProvideMoodleLoginRequest]) (*connect.Response[api.ProvideMoodleLoginResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("services.vchs.api.VchsService.ProvideMoodleLogin is not implemented"))
}

func (s Service) GetStudentData(ctx context.Context, req *connect.Request[api.GetStudentDataRequest]) (*connect.Response[api.GetStudentDataResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("services.vchs.api.VchsService.GetStudentData is not implemented"))
}
