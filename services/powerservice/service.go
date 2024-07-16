package powerservice

import (
	"context"
	"database/sql"
	"fmt"
	"vcassist-backend/lib/oauth"
	scraper "vcassist-backend/lib/scrapers/powerschool"
	"vcassist-backend/lib/timezone"
	powerschoolv1 "vcassist-backend/proto/vcassist/scrapers/powerschool/v1"
	keychainv1 "vcassist-backend/proto/vcassist/services/keychain/v1"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	powerservicev1 "vcassist-backend/proto/vcassist/services/powerservice/v1"
	studentdatav1 "vcassist-backend/proto/vcassist/services/studentdata/v1"
	"vcassist-backend/services/powerservice/db"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/protobuf/proto"
)

var tracer = otel.Tracer("services/powerschool")

const keychainNamespace = "powerservice"

type OAuthConfig struct {
	BaseLoginUrl string `json:"base_login_url"`
	RefreshUrl   string `json:"refresh_url"`
	ClientId     string `json:"client_id"`
}

type Service struct {
	baseUrl  string
	db       *sql.DB
	oauth    OAuthConfig
	qry      *db.Queries
	keychain keychainv1connect.KeychainServiceClient
}

func NewService(
	database *sql.DB,
	keychain keychainv1connect.KeychainServiceClient,
	baseUrl string,
	oauth OAuthConfig,
) Service {
	return Service{
		baseUrl:  baseUrl,
		oauth:    oauth,
		db:       database,
		qry:      db.New(database),
		keychain: keychain,
	}
}

func (s Service) GetKnownCourses(ctx context.Context, req *connect.Request[powerservicev1.GetKnownCoursesRequest]) (*connect.Response[powerservicev1.GetKnownCoursesResponse], error) {
	ctx, span := tracer.Start(ctx, "GetKnownCourses")
	defer span.End()

	courses, err := s.qry.GetKnownCourses(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to exec sql query")
		return nil, err
	}

	res := make([]*powerservicev1.KnownCourse, len(courses))
	for i, c := range courses {
		res[i] = &powerservicev1.KnownCourse{
			Guid:             c.Guid,
			Name:             c.Name,
			Period:           c.Period.String,
			Room:             c.Room.String,
			TeacherFirstName: c.Teacherfirstname.String,
			TeacherLastName:  c.Teacherlastname.String,
			TeacherEmail:     c.Teacheremail.String,
		}
	}
	return &connect.Response[powerservicev1.GetKnownCoursesResponse]{
		Msg: &powerservicev1.GetKnownCoursesResponse{
			Courses: res,
		},
	}, nil
}

func (s Service) GetAuthStatus(ctx context.Context, req *connect.Request[powerservicev1.GetAuthStatusRequest]) (*connect.Response[powerservicev1.GetAuthStatusResponse], error) {
	ctx, span := tracer.Start(ctx, "GetAuthStatus")
	defer span.End()

	studentId := req.Msg.GetStudentId()
	res, err := s.keychain.GetOAuth(ctx, &connect.Request[keychainv1.GetOAuthRequest]{
		Msg: &keychainv1.GetOAuthRequest{
			Namespace: keychainNamespace,
			Id:        studentId,
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if res.Msg.GetKey() == nil || res.Msg.GetKey().GetExpiresAt() < timezone.Now().Unix() {
		span.SetStatus(codes.Ok, "got expired token")
		return &connect.Response[powerservicev1.GetAuthStatusResponse]{
			Msg: &powerservicev1.GetAuthStatusResponse{
				IsAuthenticated: false,
			},
		}, nil
	}

	span.SetStatus(codes.Ok, "token found")
	return &connect.Response[powerservicev1.GetAuthStatusResponse]{
		Msg: &powerservicev1.GetAuthStatusResponse{
			IsAuthenticated: true,
		},
	}, nil
}

func (s Service) GetOAuthFlow(ctx context.Context, _ *connect.Request[powerservicev1.GetOAuthFlowRequest]) (*connect.Response[powerservicev1.GetOAuthFlowResponse], error) {
	ctx, span := tracer.Start(ctx, "GetAuthFlow")
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

	return &connect.Response[powerservicev1.GetOAuthFlowResponse]{
		Msg: &powerservicev1.GetOAuthFlowResponse{
			Flow: &studentdatav1.OAuthFlow{
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

func (s Service) ProvideOAuth(ctx context.Context, req *connect.Request[powerservicev1.ProvideOAuthRequest]) (*connect.Response[powerservicev1.ProvideOAuthResponse], error) {
	ctx, span := tracer.Start(ctx, "ProvideOAuth")
	defer span.End()

	client, err := scraper.NewClient(s.baseUrl)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create powerschool scraper")
		return nil, err
	}

	token := req.Msg.GetToken()

	expiresAt, err := client.LoginOAuth(ctx, token)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to login")
		return nil, fmt.Errorf("failed to login: %s", err.Error())
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to start transaction")
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

	_, err = s.keychain.SetOAuth(ctx, &connect.Request[keychainv1.SetOAuthRequest]{
		Msg: &keychainv1.SetOAuthRequest{
			Namespace: keychainNamespace,
			Id:        req.Msg.GetStudentId(),
			Key: &keychainv1.OAuthKey{
				Token:      token,
				RefreshUrl: s.oauth.RefreshUrl,
				ClientId:   s.oauth.ClientId,
				ExpiresAt:  expiresAt.Unix(),
			},
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return &connect.Response[powerservicev1.ProvideOAuthResponse]{Msg: &powerservicev1.ProvideOAuthResponse{}}, nil
}

func (s Service) GetStudentData(ctx context.Context, req *connect.Request[powerservicev1.GetStudentDataRequest]) (*connect.Response[powerservicev1.GetStudentDataResponse], error) {
	ctx, span := tracer.Start(ctx, "GetStudentData")
	defer span.End()

	studentId := req.Msg.GetStudentId()

	res, err := s.keychain.GetOAuth(ctx, &connect.Request[keychainv1.GetOAuthRequest]{
		Msg: &keychainv1.GetOAuthRequest{
			Namespace: keychainNamespace,
			Id:        studentId,
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if res.Msg.GetKey() == nil {
		err := fmt.Errorf("no oauth credentials provided")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	client, err := scraper.NewClient(s.baseUrl)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create powerschoolv1.client")
		return nil, err
	}
	_, err = client.LoginOAuth(ctx, res.Msg.GetKey().GetToken())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to login with oauth token")
		return nil, err
	}

	allStudents, err := client.GetAllStudents(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch all student data")
		return nil, err
	}
	if len(allStudents.GetStudents()) == 0 {
		err := fmt.Errorf("could not find student profile, are your powerschoolv1.credentials expired?")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	psStudent := allStudents.GetStudents()[0]
	studentData, err := client.GetStudentData(ctx, &powerschoolv1.GetStudentDataInput{
		Guid: psStudent.GetGuid(),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch student data")
		return nil, err
	}

	if studentData.GetStudent() == nil {
		span.SetStatus(codes.Ok, "student data unavailable, only returning profile...")
		return &connect.Response[powerservicev1.GetStudentDataResponse]{
			Msg: &powerservicev1.GetStudentDataResponse{Profile: psStudent},
		}, nil
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
			Guid:             c.GetGuid(),
			Name:             c.GetName(),
			Teacherfirstname: sql.NullString{String: c.GetTeacherFirstName()},
			Teacherlastname:  sql.NullString{String: c.GetTeacherLastName()},
			Teacheremail:     sql.NullString{String: c.GetTeacherEmail()},
			Period:           sql.NullString{String: c.GetPeriod()},
			Room:             sql.NullString{String: c.GetRoom()},
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

	var courseMeetingList *powerschoolv1.CourseMeetingList
	if len(guids) > 0 {
		courseMeetingList, err = client.GetCourseMeetingList(ctx, &powerschoolv1.GetCourseMeetingListInput{
			SectionGuids: guids,
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to fetch course meetings")
			return nil, err
		}
	}

	// cache and return response
	response := &powerservicev1.GetStudentDataResponse{
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
		Studentid: studentId,
		Createdat: timezone.Now().Unix(),
		Cached:    serializedResponse,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to exec sql query")
		return nil, err
	}

	return &connect.Response[powerservicev1.GetStudentDataResponse]{
		Msg: response,
	}, nil
}
