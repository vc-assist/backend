package vcsis

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"vcassist-backend/lib/gradestore"
	scraper "vcassist-backend/lib/scrapers/powerschool"
	"vcassist-backend/lib/timezone"
	keychainv1 "vcassist-backend/proto/vcassist/services/keychain/v1"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	"vcassist-backend/proto/vcassist/services/linker/v1/linkerv1connect"
	sisv1 "vcassist-backend/proto/vcassist/services/sis/v1"
	"vcassist-backend/services/auth/verifier"
	"vcassist-backend/services/vcsis/db"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

const keychainNamespace = "vcsis"

type Service struct {
	baseUrl           string
	oauth             OAuthConfig
	keychain          keychainv1connect.KeychainServiceClient
	linker            linkerv1connect.LinkerServiceClient
	gradestore        gradestore.Store
	qry               *db.Queries
	weightData        WeightData
	weightCourseNames []string
}

type ServiceOptions struct {
	Database *sql.DB
	Keychain keychainv1connect.KeychainServiceClient
	BaseUrl  string
	OAuth    OAuthConfig
}

func NewService(opts ServiceOptions) Service {
	if opts.OAuth.BaseLoginUrl == "" {
		panic("empty base login url")
	}

	s := Service{
		qry:        db.New(opts.Database),
		gradestore: gradestore.NewStore(opts.Database),
		baseUrl:    opts.BaseUrl,
		oauth:      opts.OAuth,
		keychain:   opts.Keychain,
	}

	go s.gradeSnapshotDaemon(context.Background())
	go s.preloadStudentDataDaemon(context.Background())

	return s
}

func (s Service) GetCredentialStatus(ctx context.Context, req *connect.Request[sisv1.GetCredentialStatusRequest]) (*connect.Response[sisv1.GetCredentialStatusResponse], error) {
	span := trace.SpanFromContext(ctx)
	profile, _ := verifier.ProfileFromContext(ctx)

	res, err := s.keychain.GetOAuth(ctx, &connect.Request[keychainv1.GetOAuthRequest]{
		Msg: &keychainv1.GetOAuthRequest{
			Namespace: keychainNamespace,
			Id:        profile.Email,
		},
	})
	if err != nil {
		return nil, err
	}
	if res.Msg.GetKey() == nil || res.Msg.GetKey().GetExpiresAt() < timezone.Now().Unix() {
		span.SetStatus(codes.Ok, "got expired token")
		return &connect.Response[sisv1.GetCredentialStatusResponse]{
			Msg: &sisv1.GetCredentialStatusResponse{
				Status: &keychainv1.CredentialStatus{
					Name:      "PowerSchool",
					Picture:   "",
					Provided:  false,
					LoginFlow: &keychainv1.CredentialStatus_Oauth{},
				},
			},
		}, nil
	}

	oauthFlow, err := s.oauth.GetOAuthFlow()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create oauth flow")
		return nil, err
	}

	return &connect.Response[sisv1.GetCredentialStatusResponse]{
		Msg: &sisv1.GetCredentialStatusResponse{
			Status: &keychainv1.CredentialStatus{
				Name:     "PowerSchool",
				Picture:  "",
				Provided: true,
				LoginFlow: &keychainv1.CredentialStatus_Oauth{
					Oauth: oauthFlow,
				},
			},
		},
	}, nil
}

func (s Service) ProvideCredential(ctx context.Context, req *connect.Request[sisv1.ProvideCredentialRequest]) (*connect.Response[sisv1.ProvideCredentialResponse], error) {
	profile, _ := verifier.ProfileFromContext(ctx)

	client, err := scraper.NewClient(s.baseUrl)
	if err != nil {
		return nil, err
	}
	token := req.Msg.GetToken().GetToken()
	expiresAt, err := client.LoginOAuth(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to login: %s", err.Error())
	}

	_, err = s.keychain.SetOAuth(ctx, &connect.Request[keychainv1.SetOAuthRequest]{
		Msg: &keychainv1.SetOAuthRequest{
			Namespace: keychainNamespace,
			Id:        profile.Email,
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

	return &connect.Response[sisv1.ProvideCredentialResponse]{
		Msg: &sisv1.ProvideCredentialResponse{},
	}, nil
}

func (s Service) getCachedData(ctx context.Context, studentId string) (*sisv1.GetDataResponse, error) {
	row, err := s.qry.GetStudentData(ctx, studentId)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no data cached")
	}
	if err != nil {
		return nil, err
	}
	if timezone.Now().Sub(row.LastUpdated).Hours() > 1 {
		return nil, fmt.Errorf("data is more than a day old")
	}

	data := &sisv1.GetDataResponse{}
	err = proto.Unmarshal(row.Data, data)
	return data, err
}

func (s Service) cacheNewData(ctx context.Context, studentId string, data *sisv1.GetDataResponse) error {
	marshaled, err := proto.Marshal(data)
	if err != nil {
		return err
	}
	err = s.qry.CacheStudentData(ctx, db.CacheStudentDataParams{
		StudentID: studentId,
		Data:      marshaled,
	})
	return err
}

func (s Service) addGradeSnapshots(ctx context.Context, studentId string, data *sisv1.GetDataResponse) error {
	series, err := s.gradestore.Pull(ctx, studentId)
	if err != nil {
		return err
	}

	for _, course := range series {
		var target *sisv1.CourseData
		for _, tc := range data.Courses {
			if tc.Guid == course.Course {
				target = tc
				break
			}
		}
		if target == nil {
			slog.WarnContext(ctx, "failed to find saved grade snapshot course in powerschool data", "course", course.Course)
			continue
		}

		snapshots := make([]*sisv1.GradeSnapshot, len(course.Snapshots))
		for i, s := range course.Snapshots {
			snapshots[i] = &sisv1.GradeSnapshot{
				Time:  s.Time.Unix(),
				Value: s.Value,
			}
		}
		target.Snapshots = snapshots
	}
	return nil
}

func (s Service) cachedStudentDataResponse(data *sisv1.GetDataResponse) *connect.Response[sisv1.GetDataResponse] {
	response := &connect.Response[sisv1.GetDataResponse]{
		Msg: data,
	}
	response.Header().Add("cache-control", "max-age=10800")
	return response
}

func (s Service) scrape(ctx context.Context, studentId string) (*sisv1.GetDataResponse, error) {
	res, err := s.keychain.GetOAuth(ctx, &connect.Request[keychainv1.GetOAuthRequest]{
		Msg: &keychainv1.GetOAuthRequest{
			Namespace: keychainNamespace,
			Id:        studentId,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("keychain: %w", err)
	}
	if res.Msg.GetKey() == nil {
		return nil, fmt.Errorf("no oauth credentials provided")
	}

	client, err := scraper.NewClient(s.baseUrl)
	if err != nil {
		return nil, fmt.Errorf("powerschool client constructor: %w", err)
	}
	_, err = client.LoginOAuth(ctx, res.Msg.GetKey().GetToken())
	if err != nil {
		return nil, fmt.Errorf("oauth login: %w", err)
	}

	data, err := Scrape(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("scraping: %w", err)
	}

	err = s.addGradeSnapshots(ctx, studentId, data)
	if err != nil {
		slog.WarnContext(ctx, "add grade snapshots", "err", err)
	}
	err = s.addWeights(ctx, data.Courses)
	if err != nil {
		slog.WarnContext(ctx, "add weights", "err", err)
	}

	return data, nil
}

func (s Service) GetData(ctx context.Context, req *connect.Request[sisv1.GetDataRequest]) (*connect.Response[sisv1.GetDataResponse], error) {
	profile, _ := verifier.ProfileFromContext(ctx)
	studentId := profile.Email

	cached, err := s.getCachedData(ctx, studentId)
	if err == nil {
		slog.DebugContext(ctx, "student data cache hit", "student_id", studentId)
		return s.cachedStudentDataResponse(cached), nil
	} else {
		slog.WarnContext(ctx, "get cached data", "err", err)
	}

	data, err := s.scrape(ctx, studentId)
	if err != nil {
		slog.ErrorContext(ctx, "scrape", "err", err)
		return nil, err
	}

	err = s.cacheNewData(ctx, studentId, data)
	if err != nil {
		slog.WarnContext(ctx, "cache student data response", "err", err)
	}

	return s.cachedStudentDataResponse(data), nil
}
