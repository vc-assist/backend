package powerschoolapi

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"vcassist-backend/cmd/powerschool_api/api"
	"vcassist-backend/cmd/powerschool_api/api/apiconnect"
	"vcassist-backend/cmd/powerschool_api/db"
	"vcassist-backend/lib/oauth"
	"vcassist-backend/lib/platforms/powerschool"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
)

type PowerschoolService struct {
	baseUrl string
	qry     *db.Queries
	db      *sql.DB

	oauth  OAuthConfig
	tracer trace.Tracer

	apiconnect.UnimplementedPowerschoolServiceHandler
}

func NewPowerschoolService(database *sql.DB, config Config) PowerschoolService {
	return PowerschoolService{
		baseUrl: config.BaseUrl,
		oauth:   config.OAuth,
		qry:     db.New(database),
		db:      database,
		tracer:  otel.GetTracerProvider().Tracer("service"),
	}
}

func (s PowerschoolService) GetAuthStatus(
	ctx context.Context,
	req *connect.Request[api.GetAuthStatusRequest],
) (*connect.Response[api.GetAuthStatusResponse], error) {
	ctx, span := s.tracer.Start(ctx, "service:GetAuthStatus")
	defer span.End()

	token, err := s.qry.GetOAuthToken(ctx, req.Msg.GetStudentId())
	if token.Expiresat < time.Now().Unix() {
		span.SetStatus(codes.Ok, "got expired token")

		return &connect.Response[api.GetAuthStatusResponse]{
			Msg: &api.GetAuthStatusResponse{
				IsAuthenticated: false,
			},
		}, nil
	}
	if err == sql.ErrNoRows {
		span.SetStatus(codes.Ok, "token not found")

		return &connect.Response[api.GetAuthStatusResponse]{
			Msg: &api.GetAuthStatusResponse{
				IsAuthenticated: false,
			},
		}, nil
	}
	if err != nil {
		return nil, err
	}

	span.SetStatus(codes.Ok, "token found")
	return &connect.Response[api.GetAuthStatusResponse]{
		Msg: &api.GetAuthStatusResponse{
			IsAuthenticated: token.Token != "",
		},
	}, nil
}

func (s PowerschoolService) GetAuthFlow(
	ctx context.Context,
	req *connect.Request[api.GetAuthFlowRequest],
) (*connect.Response[api.GetAuthFlowResponse], error) {
	if (s.oauth == OAuthConfig{}) {
		return nil, fmt.Errorf("non-oauth authentication is not supported yet")
	}

	codeVerifier, err := oauth.GenerateCodeVerifier()
	if err != nil {
		return nil, err
	}

	return &connect.Response[api.GetAuthFlowResponse]{
		Msg: &api.GetAuthFlowResponse{
			Flow: &api.GetAuthFlowResponse_Oauth{
				Oauth: &api.OAuthFlow{
					BaseLoginUrl:    s.oauth.BaseLoginUrl,
					AccessType:      "offline",
					Scope:           "openid email profile",
					RedirectUri:     "com.powerschool.portal://",
					CodeVerifier:    codeVerifier,
					ClientId:        s.oauth.ClientId,
					TokenRequestUrl: "https://oauth2.googleapis.com/token",
				},
			},
		},
	}, nil
}

func (s PowerschoolService) ProvideOAuth(
	ctx context.Context,
	req *connect.Request[api.ProvideOAuthRequest],
) (*connect.Response[api.ProvideOAuthResponse], error) {
	ctx, span := s.tracer.Start(ctx, "service:ProvideOAuth")
	defer span.End()

	client, err := powerschool.NewClient(s.baseUrl)
	if err != nil {
		return nil, err
	}

	expiresAt, err := client.LoginOAuth(ctx, req.Msg.GetToken())
	if err != nil {
		return &connect.Response[api.ProvideOAuthResponse]{
			Msg: &api.ProvideOAuthResponse{
				Success: false,
				Message: fmt.Sprintf("failed to login: %s", err.Error()),
			},
		}, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		// err will be set to the latest err when this function
		// is called (which is everywhere there is a return after this)
		// because of the wonders of closures
		if err != nil {
			tx.Rollback()
			return
		}
		tx.Commit()
	}()

	qry := s.qry.WithTx(tx)

	err = qry.CreateStudent(ctx, req.Msg.GetStudentId())
	if err != nil {
		return nil, err
	}
	err = qry.CreateOrUpdateOAuthToken(ctx, db.CreateOrUpdateOAuthTokenParams{
		Studentid: req.Msg.GetStudentId(),
		Token:     req.Msg.GetToken(),
		Expiresat: expiresAt.Unix(),
	})
	if err != nil {
		return nil, err
	}

	return &connect.Response[api.ProvideOAuthResponse]{
		Msg: &api.ProvideOAuthResponse{
			Success: true,
			Message: "Credentials are valid.",
		},
	}, nil
}

func (s PowerschoolService) GetStudentData(
	ctx context.Context,
	req *connect.Request[api.GetStudentDataRequest],
) (*connect.Response[api.GetStudentDataResponse], error) {
	ctx, span := s.tracer.Start(ctx, "service:GetStudentData")
	defer span.End()

	// get oauth token & login
	token, err := s.qry.GetOAuthToken(ctx, req.Msg.GetStudentId())
	if err == sql.ErrNoRows {
		// if we are to support other auth methods in the future
		// you would add additional code to handle it in this branch
		return nil, fmt.Errorf("you don't have any credentials that can request student data")
	}
	if err != nil {
		return nil, err
	}
	if token.Expiresat < time.Now().Unix() {
		return nil, fmt.Errorf("your credentials have expired, please call ProvideOAuth again")
	}
	client, err := powerschool.NewClient(s.baseUrl)
	if err != nil {
		return nil, err
	}
	_, err = client.LoginOAuth(ctx, token.Token)
	if err != nil {
		return nil, err
	}

	// fetch data
	allStudents, err := client.GetAllStudents(ctx)
	if err != nil {
		return nil, err
	}
	if len(allStudents.GetStudents()) == 0 {
		return nil, fmt.Errorf("could not find student profile, are your powerschool credentials expired?")
	}

	psStudent := allStudents.GetStudents()[0]
	studentData, err := client.GetStudentData(ctx, &powerschool.GetStudentDataInput{
		Guid: psStudent.GetGuid(),
	})
	if err != nil {
		return nil, err
	}

	if studentData.Student == nil {
		return &connect.Response[api.GetStudentDataResponse]{
			Msg: &api.GetStudentDataResponse{
				Profile: psStudent,
			},
		}, nil
	}

	courseList := studentData.GetStudent().GetSections()
	guids := make([]string, len(courseList))
	for i, course := range courseList {
		guids[i] = course.GetGuid()
	}

	courseMeetingList, err := client.GetCourseMeetingList(ctx, &powerschool.GetCourseMeetingListInput{
		SectionGuids: guids,
	})
	if err != nil {
		return nil, err
	}

	// cache and return response
	response := &api.GetStudentDataResponse{
		Profile:    psStudent,
		CourseData: courseList,
		Meetings:   courseMeetingList,
	}

	serializedResponse, err := proto.Marshal(response)
	if err != nil {
		return nil, err
	}
	err = s.qry.CreateOrUpdateStudentData(ctx, db.CreateOrUpdateStudentDataParams{
		Studentid: token.Studentid,
		Createdat: time.Now().Unix(),
		Cached:    serializedResponse,
	})
	if err != nil {
		return nil, err
	}

	return &connect.Response[api.GetStudentDataResponse]{Msg: response}, nil
}
