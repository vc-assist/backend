package main

import (
	"context"
	"vcassist-backend/cmd/authd/api"
	"vcassist-backend/cmd/authd/api/apiconnect"

	"connectrpc.com/connect"
)

type GrpcService struct {
	service Service

	apiconnect.UnimplementedAuthServiceHandler
}

func (s GrpcService) StartLogin(ctx context.Context, req *connect.Request[api.StartLoginRequest]) (*connect.Response[api.StartLoginResponse], error) {
	err := s.service.StartLogin(ctx, req.Msg.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &connect.Response[api.StartLoginResponse]{Msg: &api.StartLoginResponse{}}, nil
}

func (s GrpcService) ConsumeVerificationCode(ctx context.Context, req *connect.Request[api.ConsumeVerificationCodeRequest]) (*connect.Response[api.ConsumeVerificationCodeResponse], error) {
	token, err := s.service.ConsumeVerificationCode(ctx, req.Msg.Email, req.Msg.ProvidedCode)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &connect.Response[api.ConsumeVerificationCodeResponse]{
		Msg: &api.ConsumeVerificationCodeResponse{Token: token},
	}, nil
}

func (s GrpcService) VerifyToken(ctx context.Context, req *connect.Request[api.VerifyTokenRequest]) (*connect.Response[api.VerifyTokenResponse], error) {
	user, err := s.service.VerifyToken(ctx, req.Msg.Token)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &connect.Response[api.VerifyTokenResponse]{
		Msg: &api.VerifyTokenResponse{Email: user.Email},
	}, nil
}
