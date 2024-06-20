package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"vcassist-backend/cmd/powerschool_api/api"
	"vcassist-backend/cmd/powerschool_api/api/apiconnect"
	"vcassist-backend/cmd/powerschool_api/db"
	"vcassist-backend/lib/platforms/powerschool"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
)

type PowerschoolService struct {
	baseUrl string
	qry     *db.Queries
	db      *sql.DB

	meter  metric.Meter
	tracer trace.Tracer

	apiconnect.UnimplementedPowerschoolServiceHandler
}

func (s PowerschoolService) GetOAuth(
	ctx context.Context,
	req *connect.Request[api.GetOAuthRequest],
) (*connect.Response[api.GetOAuthResponse], error) {
	ctx, span := s.tracer.Start(ctx, "service:getOAuth")
	defer span.End()

	token, err := s.qry.GetOAuthToken(ctx, req.Msg.StudentId)
	if token.Expiresat < time.Now().Unix() {
		span.SetStatus(codes.Ok, "got expired token")

		return &connect.Response[api.GetOAuthResponse]{
			Msg: &api.GetOAuthResponse{
				HasToken: false,
			},
		}, nil
	}
	if err == sql.ErrNoRows {
		span.SetStatus(codes.Ok, "token not found")

		return &connect.Response[api.GetOAuthResponse]{
			Msg: &api.GetOAuthResponse{
				HasToken: false,
			},
		}, nil
	}
	if err != nil {
		return nil, err
	}

	span.SetStatus(codes.Ok, "token found")
	return &connect.Response[api.GetOAuthResponse]{
		Msg: &api.GetOAuthResponse{
			HasToken: token.Token != "",
		},
	}, nil
}

func (s PowerschoolService) ProvideOAuth(
	ctx context.Context,
	req *connect.Request[api.ProvideOAuthRequest],
) (*connect.Response[api.ProvideOAuthResponse], error) {
	ctx, span := s.tracer.Start(ctx, "service:provideOAuth")
	defer span.End()

	client, err := powerschool.NewClient(s.baseUrl)
	if err != nil {
		return nil, err
	}

	expiresAt, err := client.LoginOAuth(ctx, req.Msg.Token)
	if err != nil {
		return &connect.Response[api.ProvideOAuthResponse]{
			Msg: &api.ProvideOAuthResponse{
				Success: false,
				Message: fmt.Sprintf("failed to login: %s", err.Error()),
			},
		}, nil
	}
	_, err = client.GetAllStudents(ctx)
	if err != nil {
		return &connect.Response[api.ProvideOAuthResponse]{
			Msg: &api.ProvideOAuthResponse{
				Success: false,
				Message: fmt.Sprintf("session invalid: %s", err.Error()),
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

	err = qry.CreateStudent(ctx, req.Msg.StudentId)
	if err != nil {
		return nil, err
	}
	err = qry.CreateOrUpdateOAuthToken(ctx, db.CreateOrUpdateOAuthTokenParams{
		Studentid: req.Msg.StudentId,
		Token:     req.Msg.Token,
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
	ctx, span := s.tracer.Start(ctx, "service:getStudentData")
	defer span.End()

	// get oauth token & login
	token, err := s.qry.GetOAuthToken(ctx, req.Msg.StudentId)
	if err != nil {
		// if we are to support other auth methods in the future
		// you would add additional code to handle it in the
		// if err == sql.ErrNoRows ... branch
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
	if len(allStudents.Students) == 0 {
		return nil, fmt.Errorf("could not find student profile, are your powerschool credentials expired?")
	}

	psStudent := allStudents.Students[0]
	studentData, err := client.GetStudentData(ctx, &powerschool.GetStudentDataInput{
		Guid: psStudent.Guid,
	})
	if err != nil {
		return nil, err
	}
	courseList := studentData.Student.Sections

	guids := make([]string, len(courseList))
	for i, course := range courseList {
		guids[i] = course.Guid
	}

	courseMeetingList, err := client.GetCourseMeetingList(ctx, &powerschool.GetCourseMeetingListInput{
		SectionGuids: guids,
	})
	if err != nil {
		return nil, err
	}

	// cache and return response
	response := &api.GetStudentDataResponse{
		Profile:    allStudents.Students[0],
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
