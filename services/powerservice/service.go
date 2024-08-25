package powerservice

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"
	"vcassist-backend/lib/oauth"
	scraper "vcassist-backend/lib/scrapers/powerschool"
	"vcassist-backend/lib/timezone"
	keychainv1 "vcassist-backend/proto/vcassist/services/keychain/v1"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	powerservicev1 "vcassist-backend/proto/vcassist/services/powerservice/v1"
	studentdatav1 "vcassist-backend/proto/vcassist/services/studentdata/v1"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

var tracer = otel.Tracer("vcassist.services.powerservice")

const keychainNamespace = "powerservice"

type OAuthConfig struct {
	BaseLoginUrl string
	RefreshUrl   string
	ClientId     string
}

type Service struct {
	baseUrl  string
	oauth    OAuthConfig
	keychain keychainv1connect.KeychainServiceClient
}

func NewService(
	keychain keychainv1connect.KeychainServiceClient,
	baseUrl string,
	oauth OAuthConfig,
) Service {
	return Service{
		baseUrl:  baseUrl,
		oauth:    oauth,
		keychain: keychain,
	}
}

func (s Service) GetAuthStatus(ctx context.Context, req *connect.Request[powerservicev1.GetAuthStatusRequest]) (*connect.Response[powerservicev1.GetAuthStatusResponse], error) {
	span := trace.SpanFromContext(ctx)

	studentId := req.Msg.GetStudentId()
	res, err := s.keychain.GetOAuth(ctx, &connect.Request[keychainv1.GetOAuthRequest]{
		Msg: &keychainv1.GetOAuthRequest{
			Namespace: keychainNamespace,
			Id:        studentId,
		},
	})
	if err != nil {
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
	if (s.oauth == OAuthConfig{}) {
		err := fmt.Errorf("non-oauth authentication is not supported yet")
		return nil, err
	}

	codeVerifier, err := oauth.GenerateCodeVerifier()
	if err != nil {
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
	client, err := scraper.NewClient(s.baseUrl)
	if err != nil {
		return nil, err
	}

	token := req.Msg.GetToken()

	expiresAt, err := client.LoginOAuth(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to login: %s", err.Error())
	}

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
		return nil, err
	}

	return &connect.Response[powerservicev1.ProvideOAuthResponse]{Msg: &powerservicev1.ProvideOAuthResponse{}}, nil
}

func getCurrentWeek() (start time.Time, stop time.Time) {
	now := timezone.Now()
	start = now.Add(-time.Hour * 24 * time.Duration(now.Weekday()))
	stop = now.Add(time.Hour * 24 * time.Duration(time.Saturday-now.Weekday()))
	return start, stop
}

func (s Service) GetStudentData(ctx context.Context, req *connect.Request[powerservicev1.GetStudentDataRequest]) (*connect.Response[powerservicev1.GetStudentDataResponse], error) {
	span := trace.SpanFromContext(ctx)

	studentId := req.Msg.GetStudentId()

	res, err := s.keychain.GetOAuth(ctx, &connect.Request[keychainv1.GetOAuthRequest]{
		Msg: &keychainv1.GetOAuthRequest{
			Namespace: keychainNamespace,
			Id:        studentId,
		},
	})
	if err != nil {
		return nil, err
	}
	if res.Msg.GetKey() == nil {
		err := fmt.Errorf("no oauth credentials provided")
		return nil, err
	}

	client, err := scraper.NewClient(s.baseUrl)
	if err != nil {
		return nil, err
	}
	_, err = client.LoginOAuth(ctx, res.Msg.GetKey().GetToken())
	if err != nil {
		return nil, err
	}

	allStudents, err := client.GetAllStudents(ctx)
	if err != nil {
		return nil, err
	}
	if len(allStudents.Profiles) == 0 {
		err := fmt.Errorf("could not find student profile, are your credentials expired?")
		return nil, err
	}

	psStudent := allStudents.Profiles[0]
	studentData, err := client.GetStudentData(ctx, scraper.GetStudentDataRequest{
		Guid: psStudent.Guid,
	})
	if err != nil {
		return nil, err
	}

	// MAY BE USED LATER, DO NOT DELETE
	// studentPhoto, err := client.GetStudentPhoto(ctx, scraper.GetStudentPhotoRequest{
	// 	Guid: psStudent.Guid,
	// })
	// if err != nil {
	// 	span.RecordError(err)
	// 	span.SetStatus(codes.Error, "failed to get student photo")
	// }

	gpa, err := strconv.ParseFloat(psStudent.CurrentGpa, 32)
	if err != nil {
		slog.Warn("failed to parse gpa", "gpa", psStudent.CurrentGpa, "err", err)
	}

	if len(studentData.Student.Courses) == 0 {
		span.SetStatus(codes.Error, "student data unavailable, only returning profile...")
		return &connect.Response[powerservicev1.GetStudentDataResponse]{
			Msg: &powerservicev1.GetStudentDataResponse{Profile: &powerservicev1.StudentProfile{
				Guid:       psStudent.Guid,
				CurrentGpa: float32(gpa),
				FirstName:  psStudent.FirstName,
				LastName:   psStudent.LastName,
				// photo is disabled for now as it doesn't have a use
				// Photo: "",
			}},
		}, nil
	}

	courses := transformCourses(ctx, studentData.Student.Courses)

	if len(courses) > 0 {
		guids := make([]string, len(courses))
		for i, course := range courses {
			guids[i] = course.Guid
		}

		start, stop := getCurrentWeek()
		res, err := client.GetCourseMeetingList(ctx, scraper.GetCourseMeetingListRequest{
			CourseGuids: guids,
			Start:       start.Format(time.RFC3339),
			Stop:        stop.Format(time.RFC3339),
		})
		if err != nil {
			slog.Warn(
				"failed to fetch course meetings",
				"err", err,
			)
		}

		transformCourseMeetings(ctx, courses, res.Meetings)
	}

	schools := transformSchools(psStudent.Schools)
	bulletins := transformBulletins(psStudent.Bulletins)

	return &connect.Response[powerservicev1.GetStudentDataResponse]{
		Msg: &powerservicev1.GetStudentDataResponse{
			Profile: &powerservicev1.StudentProfile{
				Guid:       psStudent.Guid,
				CurrentGpa: float32(gpa),
				FirstName:  psStudent.FirstName,
				LastName:   psStudent.LastName,
			},
			Schools:   schools,
			Bulletins: bulletins,
			Courses:   courses,
		},
	}, nil
}
