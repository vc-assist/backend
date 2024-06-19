package powerschool_api

import (
	"context"
	"vcassist-backend/cmd/powerschool_api/pb"

	"connectrpc.com/connect"
)

type PowerschoolService struct {

}

func (s PowerschoolService) ProvideOAuth(ctx context.Context, req *connect.Request[pb.ProvideOAuthRequest]) {

}

