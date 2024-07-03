package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"vcassist-backend/cmd/powerschoold/api"
	"vcassist-backend/cmd/powerschoold/db"
	"vcassist-backend/lib/oauth"
	"vcassist-backend/lib/platforms/powerschool"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/protobuf/proto"
)

var tracer = otel.Tracer("powerschoold/service")

type Service struct {
	baseUrl string
	qry     *db.Queries
	db      *sql.DB
	oauth   OAuthConfig
}

func NewService(database *sql.DB, config Config) Service {
	return Service{
		baseUrl: config.BaseUrl,
		oauth:   config.OAuth,
		qry:     db.New(database),
		db:      database,
	}
}

func (s Service) GetKnownCourses(ctx context.Context) ([]*api.KnownCourse, error) {
	ctx, span := tracer.Start(ctx, "service:GetKnownCourses")
	defer span.End()

	courses, err := s.qry.GetKnownCourses(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to exec sql query")
		return nil, err
	}

	res := make([]*api.KnownCourse, len(courses))
	for i, c := range courses {
		res[i] = &api.KnownCourse{
			Guid:             c.Guid,
			Name:             c.Name,
			Period:           c.Period.String,
			Room:             c.Room.String,
			TeacherFirstName: c.Teacherfirstname.String,
			TeacherLastName:  c.Teacherlastname.String,
			TeacherEmail:     c.Teacheremail.String,
		}
	}
	return res, nil
}

func (s Service) GetAuthStatus(ctx context.Context, studentId string) (bool, error) {
	ctx, span := tracer.Start(ctx, "service:GetAuthStatus")
	defer span.End()

	token, err := s.qry.GetOAuthToken(ctx, studentId)
	if token.Expiresat < time.Now().Unix() {
		span.SetStatus(codes.Ok, "got expired token")

		return false, nil
	}
	if err == sql.ErrNoRows {
		span.SetStatus(codes.Ok, "token not found")

		return false, nil
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to exec sql query")
		return false, connect.NewError(connect.CodeInternal, err)
	}

	span.SetStatus(codes.Ok, "token found")
	return token.Token != "", nil
}

func (s Service) GetAuthFlow(ctx context.Context) (*api.GetAuthFlowResponse, error) {
	ctx, span := tracer.Start(ctx, "service:GetAuthFlow")
	defer span.End()

	if (s.oauth == OAuthConfig{}) {
		err := fmt.Errorf("non-oauth authentication is not supported yet")
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	codeVerifier, err := oauth.GenerateCodeVerifier()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to generate code verifier")
		return nil, err
	}

	return &api.GetAuthFlowResponse{
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
	}, nil
}

func (s Service) ProvideOAuth(ctx context.Context, studentId, token string) error {
	ctx, span := tracer.Start(ctx, "service:ProvideOAuth")
	defer span.End()

	client, err := powerschool.NewClient(s.baseUrl)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create powerschool client")
		return err
	}

	expiresAt, err := client.LoginOAuth(ctx, token)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to login")
		return fmt.Errorf("failed to login: %s", err.Error())
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to start transaction")
		return err
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

	err = qry.CreateStudent(ctx, studentId)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to exec sql query (1)")
		return err
	}
	err = qry.CreateOrUpdateOAuthToken(ctx, db.CreateOrUpdateOAuthTokenParams{
		Studentid: studentId,
		Token:     token,
		Expiresat: expiresAt.Unix(),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to exec sql query (2)")
		return err
	}

	return nil
}

var NoCredentialsErr = fmt.Errorf("you don't have any credentials that can request student data")
var ExpiredCredentialsErr = fmt.Errorf("your credentials have expired, please call ProvideOAuth again")

func (s Service) GetStudentData(ctx context.Context, studentId string) (*api.GetStudentDataResponse, error) {
	ctx, span := tracer.Start(ctx, "service:GetStudentData")
	defer span.End()

	// get oauth token & login
	token, err := s.qry.GetOAuthToken(ctx, studentId)
	if err == sql.ErrNoRows {
		// if we are to support other auth methods in the future
		// you would add additional code to handle it in this branch
		span.SetStatus(codes.Error, NoCredentialsErr.Error())
		return nil, NoCredentialsErr
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to exec sql query")
		return nil, err
	}
	if token.Expiresat < time.Now().Unix() {
		span.SetStatus(codes.Error, ExpiredCredentialsErr.Error())
		return nil, ExpiredCredentialsErr
	}

	client, err := powerschool.NewClient(s.baseUrl)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create powerschool client")
		return nil, err
	}
	_, err = client.LoginOAuth(ctx, token.Token)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to login with oauth token")
		return nil, err
	}

	// fetch data
	allStudents, err := client.GetAllStudents(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch all student data")
		return nil, err
	}
	if len(allStudents.GetStudents()) == 0 {
		err := fmt.Errorf("could not find student profile, are your powerschool credentials expired?")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	psStudent := allStudents.GetStudents()[0]
	studentData, err := client.GetStudentData(ctx, &powerschool.GetStudentDataInput{
		Guid: psStudent.GetGuid(),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch student data")
		return nil, err
	}

	if studentData.Student == nil {
		span.SetStatus(codes.Ok, "student data unavailable, only returning profile...")
		return &api.GetStudentDataResponse{Profile: psStudent}, nil
	}

	courseList := studentData.GetStudent().GetSections()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to begin transaction")
	}
	defer tx.Rollback()
	txqry := s.qry.WithTx(tx)
	for _, c := range courseList {
		err = txqry.CreateOrUpdateKnownCourse(ctx, db.CreateOrUpdateKnownCourseParams{
			Guid:             c.Guid,
			Name:             c.Name,
			Teacherfirstname: sql.NullString{String: c.TeacherFirstName},
			Teacherlastname:  sql.NullString{String: c.TeacherLastName},
			Teacheremail:     sql.NullString{String: c.TeacherEmail},
			Period:           sql.NullString{String: c.Period},
			Room:             sql.NullString{String: c.Room},
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to add known course to register transaction")
		}
	}
	err = tx.Commit()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to register known courses")
	}

	guids := make([]string, len(courseList))
	for i, course := range courseList {
		guids[i] = course.GetGuid()
	}

	var courseMeetingList *powerschool.CourseMeetingList
	if len(guids) > 0 {
		courseMeetingList, err = client.GetCourseMeetingList(ctx, &powerschool.GetCourseMeetingListInput{
			SectionGuids: guids,
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to fetch course meetings")
			return nil, err
		}
	}

	// cache and return response
	response := &api.GetStudentDataResponse{
		Profile:    psStudent,
		CourseData: courseList,
		Meetings:   courseMeetingList,
	}

	serializedResponse, err := proto.Marshal(response)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to cache student data")
		return nil, err
	}
	err = s.qry.CreateOrUpdateStudentData(ctx, db.CreateOrUpdateStudentDataParams{
		Studentid: token.Studentid,
		Createdat: time.Now().Unix(),
		Cached:    serializedResponse,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to exec sql query")
		return nil, err
	}

	return response, nil
}
