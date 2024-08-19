package powerservice

import (
	"context"
	"fmt"
	"time"
	"vcassist-backend/lib/oauth"
	scraper "vcassist-backend/lib/scrapers/powerschool"
	"vcassist-backend/lib/timezone"
	powerschoolv1 "vcassist-backend/proto/vcassist/scrapers/powerschool/v1"
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
	if len(allStudents.GetStudents()) == 0 {
		err := fmt.Errorf("could not find student profile, are your powerschoolv1.credentials expired?")
		return nil, err
	}

	psStudent := allStudents.GetStudents()[0]
	studentData, err := client.GetStudentData(ctx, &powerschoolv1.GetStudentDataInput{
		Guid: psStudent.GetGuid(),
	})
	if err != nil {
		return nil, err
	}

	studentPhoto, err := client.GetStudentPhoto(ctx, &powerschoolv1.GetStudentDataInput{
		Guid: psStudent.GetGuid(),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get student photo")
	}

	if studentData.GetStudent() == nil {
		span.SetStatus(codes.Error, "student data unavailable, only returning profile...")
		return &connect.Response[powerservicev1.GetStudentDataResponse]{
			Msg: &powerservicev1.GetStudentDataResponse{Profile: psStudent},
		}, nil
	}

	courseList := studentData.GetStudent().GetSections()

	guids := make([]string, len(courseList))
	for i, course := range courseList {
		guids[i] = course.GetGuid()
	}

	var courseMeetingList *powerschoolv1.CourseMeetingList
	if len(guids) > 0 {
		start, stop := getCurrentWeek()

		courseMeetingList, err = client.GetCourseMeetingList(ctx, &powerschoolv1.GetCourseMeetingListInput{
			SectionGuids: guids,
			Start:        start.Format(time.RFC3339),
			Stop:         stop.Format(time.RFC3339),
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to fetch course meetings")
			return nil, err
		}
	}

	return &connect.Response[powerservicev1.GetStudentDataResponse]{
		Msg: &powerservicev1.GetStudentDataResponse{
			Profile:    psStudent,
			CourseData: courseList,
			Meetings:   courseMeetingList,
			Photo:      studentPhoto,
		},
	}, nil
}
