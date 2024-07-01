package main

import (
	"context"
	"vcassist-backend/cmd/powerschoold/api"
	"vcassist-backend/cmd/powerschoold/api/apiconnect"

	"connectrpc.com/connect"
)

type GrpcService struct {
	service Service

	apiconnect.UnimplementedPowerschoolServiceHandler
}

func (s GrpcService) GetAuthStatus(ctx context.Context, req *connect.Request[api.GetAuthStatusRequest]) (*connect.Response[api.GetAuthStatusResponse], error) {
	authenticated, err := s.service.GetAuthStatus(ctx, req.Msg.StudentId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &connect.Response[api.GetAuthStatusResponse]{Msg: &api.GetAuthStatusResponse{IsAuthenticated: authenticated}}, nil
}

func (s GrpcService) GetAuthFlow(ctx context.Context, _ *connect.Request[api.GetAuthFlowRequest]) (*connect.Response[api.GetAuthFlowResponse], error) {
	flow, err := s.service.GetAuthFlow(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &connect.Response[api.GetAuthFlowResponse]{Msg: flow}, nil
}

func (s GrpcService) ProvideOAuth(ctx context.Context, req *connect.Request[api.ProvideOAuthRequest]) (*connect.Response[api.ProvideOAuthResponse], error) {
	err := s.service.ProvideOAuth(ctx, req.Msg.StudentId, req.Msg.Token)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &connect.Response[api.ProvideOAuthResponse]{
		Msg: &api.ProvideOAuthResponse{
			Success: true,
			Message: "Credentials provided successfully.",
		},
	}, nil
}

func (s GrpcService) GetStudentData(ctx context.Context, req *connect.Request[api.GetStudentDataRequest]) (*connect.Response[api.GetStudentDataResponse], error) {
	data, err := s.service.GetStudentData(ctx, req.Msg.StudentId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &connect.Response[api.GetStudentDataResponse]{Msg: data}, nil
}
