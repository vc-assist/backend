package server

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/scrapers/moodle/view"
	keychainv1 "vcassist-backend/proto/vcassist/services/keychain/v1"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	vcmoodlev1 "vcassist-backend/proto/vcassist/services/vcmoodle/v1"
	"vcassist-backend/services/auth/verifier"
	"vcassist-backend/services/vcmoodle/db"

	"connectrpc.com/connect"
	"github.com/hashicorp/golang-lru/v2/expirable"

	_ "modernc.org/sqlite"
)

const baseUrl = "https://learn.vcs.net"
const keychainNamespace = "vcmoodle"

type Service struct {
	keychain        keychainv1connect.KeychainServiceClient
	qry             *db.Queries
	userCourseCache *expirable.LRU[string, []db.Course]
	coursesCache    *expirable.LRU[string, []*vcmoodlev1.Course]
}

func NewService(keychain keychainv1connect.KeychainServiceClient, data *sql.DB) Service {
	return Service{
		keychain: keychain,
		qry:      db.New(data),
		// reevaluate course list every day
		userCourseCache: expirable.NewLRU[string, []db.Course](2048, nil, time.Hour*24),
		// reevaluate course data every 12 hours
		coursesCache: expirable.NewLRU[string, []*vcmoodlev1.Course](2048, nil, time.Hour*12),
	}
}

func (s Service) GetAuthStatus(ctx context.Context, req *connect.Request[vcmoodlev1.GetAuthStatusRequest]) (*connect.Response[vcmoodlev1.GetAuthStatusResponse], error) {
	profile := verifier.ProfileFromContext(ctx)

	existing, err := s.keychain.GetUsernamePassword(ctx, &connect.Request[keychainv1.GetUsernamePasswordRequest]{
		Msg: &keychainv1.GetUsernamePasswordRequest{
			Namespace: keychainNamespace,
			Id:        profile.Email,
		},
	})
	if err != nil {
		return nil, err
	}

	return &connect.Response[vcmoodlev1.GetAuthStatusResponse]{
		Msg: &vcmoodlev1.GetAuthStatusResponse{
			Provided: existing.Msg.GetKey() != nil,
		},
	}, nil
}

func (s Service) ProvideUsernamePassword(ctx context.Context, req *connect.Request[vcmoodlev1.ProvideUsernamePasswordRequest]) (*connect.Response[vcmoodlev1.ProvideUsernamePasswordResponse], error) {
	profile := verifier.ProfileFromContext(ctx)

	_, err := s.keychain.SetUsernamePassword(ctx, &connect.Request[keychainv1.SetUsernamePasswordRequest]{
		Msg: &keychainv1.SetUsernamePasswordRequest{
			Namespace: keychainNamespace,
			Id:        profile.Email,
			Key: &keychainv1.UsernamePasswordKey{
				Username: req.Msg.GetUsername(),
				Password: req.Msg.GetPassword(),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	client, err := core.NewClient(ctx, core.ClientOptions{
		BaseUrl: baseUrl,
	})
	if err != nil {
		return nil, err
	}
	err = client.LoginUsernamePassword(ctx, req.Msg.GetUsername(), req.Msg.GetPassword())
	if err != nil {
		return nil, err
	}

	return &connect.Response[vcmoodlev1.ProvideUsernamePasswordResponse]{Msg: &vcmoodlev1.ProvideUsernamePasswordResponse{}}, nil
}

func pbResourceType(resourceType int64) vcmoodlev1.ResourceType {
	switch resourceType {
	case 0:
		return vcmoodlev1.ResourceType_GENERIC_URL
	case 1:
		return vcmoodlev1.ResourceType_BOOK
	case 2:
		return vcmoodlev1.ResourceType_HTML_AREA
	default:
		return -1
	}
}

func (s Service) getUserCourses(ctx context.Context, email string) ([]db.Course, error) {
	cached, hit := s.userCourseCache.Get(email)
	if hit {
		return cached, nil
	}

	res, err := s.keychain.GetUsernamePassword(ctx, &connect.Request[keychainv1.GetUsernamePasswordRequest]{
		Msg: &keychainv1.GetUsernamePasswordRequest{
			Namespace: keychainNamespace,
			Id:        email,
		},
	})
	if err != nil {
		return nil, err
	}

	coreClient, err := core.NewClient(ctx, core.ClientOptions{
		BaseUrl: baseUrl,
	})
	if err != nil {
		return nil, err
	}
	err = coreClient.LoginUsernamePassword(ctx, res.Msg.GetKey().GetUsername(), res.Msg.GetKey().GetPassword())
	if err != nil {
		return nil, err
	}
	client, err := view.NewClient(ctx, coreClient)
	if err != nil {
		return nil, err
	}
	courses, err := client.Courses(ctx)
	if err != nil {
		return nil, err
	}

	var courseIds []int64
	for _, c := range courses {
		id, err := c.Id()
		if err != nil {
			slog.WarnContext(ctx, "get courses: get course id", "url", c.Url.String(), "err", err)
			continue
		}
		courseIds = append(courseIds, int64(id))
	}
	dbCourses, err := s.qry.GetCourses(ctx, courseIds)
	if err != nil {
		return nil, err
	}

	evicted := s.userCourseCache.Add(email, dbCourses)
	if evicted {
		slog.WarnContext(ctx, "userCourse cache could not be added: evicted", "email", email)
	}

	return dbCourses, nil
}

func (s Service) GetCourses(ctx context.Context, req *connect.Request[vcmoodlev1.GetCoursesRequest]) (*connect.Response[vcmoodlev1.GetCoursesResponse], error) {
	profile := verifier.ProfileFromContext(ctx)

	cached, hit := s.coursesCache.Get(profile.Email)
	if hit {
		return &connect.Response[vcmoodlev1.GetCoursesResponse]{
			Msg: &vcmoodlev1.GetCoursesResponse{
				Courses: cached,
			},
		}, nil
	}

	dbCourses, err := s.getUserCourses(ctx, profile.Email)
	if err != nil {
		return nil, fmt.Errorf("getUserCourses: %w", err)
	}
	outCourses, err := GetCourseData(ctx, s.qry, dbCourses)
	if err != nil {
		return nil, err
	}

	evicted := s.coursesCache.Add(profile.Email, outCourses)
	if evicted {
		slog.WarnContext(ctx, "courses cache could not be added: evicted", "email", profile.Email)
	}

	return &connect.Response[vcmoodlev1.GetCoursesResponse]{
		Msg: &vcmoodlev1.GetCoursesResponse{
			Courses: outCourses,
		},
	}, nil
}

func (s Service) RefreshCourses(ctx context.Context, req *connect.Request[vcmoodlev1.RefreshCoursesRequest]) (*connect.Response[vcmoodlev1.RefreshCoursesResponse], error) {
	profile := verifier.ProfileFromContext(ctx)

	dbCourses, err := s.getUserCourses(ctx, profile.Email)
	if err != nil {
		return nil, fmt.Errorf("getUserCourses: %w", err)
	}
	outCourses, err := GetCourseData(ctx, s.qry, dbCourses)
	if err != nil {
		return nil, err
	}

	evicted := s.coursesCache.Add(profile.Email, outCourses)
	if evicted {
		slog.WarnContext(ctx, "courses cache could not be added: evicted", "email", profile.Email)
	}

	return &connect.Response[vcmoodlev1.RefreshCoursesResponse]{
		Msg: &vcmoodlev1.RefreshCoursesResponse{
			Courses: outCourses,
		},
	}, nil
}

func (s Service) GetChapterContent(ctx context.Context, req *connect.Request[vcmoodlev1.GetChapterContentRequest]) (*connect.Response[vcmoodlev1.GetChapterContentResponse], error) {
	content, err := s.qry.GetChapterContent(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	return &connect.Response[vcmoodlev1.GetChapterContentResponse]{
		Msg: &vcmoodlev1.GetChapterContentResponse{
			Html: content,
		},
	}, nil
}
