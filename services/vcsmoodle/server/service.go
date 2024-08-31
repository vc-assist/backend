package server

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/scrapers/moodle/view"
	keychainv1 "vcassist-backend/proto/vcassist/services/keychain/v1"
	"vcassist-backend/proto/vcassist/services/keychain/v1/keychainv1connect"
	vcsmoodlev1 "vcassist-backend/proto/vcassist/services/vcsmoodle/v1"
	"vcassist-backend/services/auth/verifier"
	"vcassist-backend/services/vcsmoodle/db"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

var tracer = otel.Tracer("vcassist.services.vcsmoodle")

const baseUrl = "https://learn.vcs.net"
const keychainNamespace = "vcsmoodle"

type Service struct {
	keychain keychainv1connect.KeychainServiceClient
	qry      *db.Queries
}

func NewService(keychain keychainv1connect.KeychainServiceClient, data *sql.DB) Service {
	return Service{
		keychain: keychain,
		qry:      db.New(data),
	}
}

func (s Service) GetAuthStatus(ctx context.Context, req *connect.Request[vcsmoodlev1.GetAuthStatusRequest]) (*connect.Response[vcsmoodlev1.GetAuthStatusResponse], error) {
	profile, _ := verifier.ProfileFromContext(ctx)

	existing, err := s.keychain.GetUsernamePassword(ctx, &connect.Request[keychainv1.GetUsernamePasswordRequest]{
		Msg: &keychainv1.GetUsernamePasswordRequest{
			Namespace: keychainNamespace,
			Id:        profile.Email,
		},
	})
	if err != nil {
		return nil, err
	}

	return &connect.Response[vcsmoodlev1.GetAuthStatusResponse]{
		Msg: &vcsmoodlev1.GetAuthStatusResponse{
			Provided: existing.Msg.GetKey() != nil,
		},
	}, nil
}

func (s Service) ProvideUsernamePassword(ctx context.Context, req *connect.Request[vcsmoodlev1.ProvideUsernamePasswordRequest]) (*connect.Response[vcsmoodlev1.ProvideUsernamePasswordResponse], error) {
	profile, _ := verifier.ProfileFromContext(ctx)

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

	return &connect.Response[vcsmoodlev1.ProvideUsernamePasswordResponse]{Msg: &vcsmoodlev1.ProvideUsernamePasswordResponse{}}, nil
}

func pbResourceType(resourceType int) vcsmoodlev1.ResourceType {
	switch resourceType {
	case 0:
		return vcsmoodlev1.ResourceType_GENERIC_URL
	case 1:
		return vcsmoodlev1.ResourceType_BOOK
	case 2:
		return vcsmoodlev1.ResourceType_HTML_AREA
	default:
		return -1
	}
}

func (s Service) GetCourses(ctx context.Context, req *connect.Request[vcsmoodlev1.GetCoursesRequest]) (*connect.Response[vcsmoodlev1.GetCoursesResponse], error) {
	profile, _ := verifier.ProfileFromContext(ctx)

	res, err := s.keychain.GetUsernamePassword(ctx, &connect.Request[keychainv1.GetUsernamePasswordRequest]{
		Msg: &keychainv1.GetUsernamePasswordRequest{
			Namespace: keychainNamespace,
			Id:        profile.Email,
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

	var courseIds []int
	for _, c := range courses {
		id, err := c.Id()
		if err != nil {
			slog.WarnContext(ctx, "failed to get course id", "url", c.Url.String())
			continue
		}
		courseIds = append(courseIds, int(id))
	}
	dbCourses, err := s.qry.GetCourses(ctx, courseIds)
	if err != nil {
		return nil, err
	}

	outCourses := make([]*vcsmoodlev1.Course, len(dbCourses))
	for i, course := range dbCourses {
		outSections := make([]*vcsmoodlev1.Section, len(course.Section))
		for i, section := range course.Section {
			outResources := make([]*vcsmoodlev1.Resource, len(section.Resource))
			for i, resource := range section.Resource {
				outChapters := make([]*vcsmoodlev1.Chapter, len(resource.Chapter))
				for i, chapter := range resource.Chapter {
					outChapters[i] = &vcsmoodlev1.Chapter{
						Id:   int64(chapter.Id),
						Name: chapter.Name,
					}
				}

				resourceType := pbResourceType(resource.Type)
				if resourceType < 0 {
					slog.WarnContext(ctx, "unknown resource type", "type", resource.Type)
					continue
				}

				outResources[i] = &vcsmoodlev1.Resource{
					Idx:            int64(resource.Idx),
					Type:           resourceType,
					Url:            resource.Url,
					DisplayContent: resource.Displaycontent,
					Chapters:       outChapters,
				}
			}

			outSections[i] = &vcsmoodlev1.Section{
				Name:      section.Name,
				Idx:       int64(section.Idx),
				Url:       fmt.Sprintf("https://learn.vcs.net/course/view.php?id=%d&section=%d", course.Id, section.Idx),
				Resources: outResources,
			}
		}
		outCourses[i] = &vcsmoodlev1.Course{
			Id:       int64(course.Id),
			Name:     course.Name,
			Url:      fmt.Sprintf("https://learn.vcs.net/course/view.php?id=%d", course.Id),
			Sections: outSections,
		}
	}

	response := &connect.Response[vcsmoodlev1.GetCoursesResponse]{
		Msg: &vcsmoodlev1.GetCoursesResponse{
			Courses: outCourses,
		},
	}
	response.Header().Add("cache-control", "max-age=10800")
	return response, nil
}

func (s Service) GetChapterContent(ctx context.Context, req *connect.Request[vcsmoodlev1.GetChapterContentRequest]) (*connect.Response[vcsmoodlev1.GetChapterContentResponse], error) {
	content, err := s.qry.GetCourseContent(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	return &connect.Response[vcsmoodlev1.GetChapterContentResponse]{
		Msg: &vcsmoodlev1.GetChapterContentResponse{
			Html: content,
		},
	}, nil
}
