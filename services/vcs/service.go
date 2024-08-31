package vcs

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/lib/timezone"
	gradesnapshotsv1 "vcassist-backend/proto/vcassist/services/gradesnapshots/v1"
	"vcassist-backend/proto/vcassist/services/gradesnapshots/v1/gradesnapshotsv1connect"
	"vcassist-backend/proto/vcassist/services/linker/v1/linkerv1connect"
	powerservicev1 "vcassist-backend/proto/vcassist/services/powerservice/v1"
	"vcassist-backend/proto/vcassist/services/powerservice/v1/powerservicev1connect"
	studentdatav1 "vcassist-backend/proto/vcassist/services/studentdata/v1"
	vcsmoodlev1 "vcassist-backend/proto/vcassist/services/vcsmoodle/v1"
	"vcassist-backend/proto/vcassist/services/vcsmoodle/v1/vcsmoodlev1connect"
	"vcassist-backend/services/auth/verifier"
	"vcassist-backend/services/vcs/db"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

var tracer = telemetry.Tracer("vcassist.services.vcs")

type WeightData = map[string]map[string]float32

type Options struct {
	Gradesnapshots       gradesnapshotsv1connect.GradeSnapshotsServiceClient
	Powerschool          powerservicev1connect.PowerschoolServiceClient
	Moodle               vcsmoodlev1connect.MoodleServiceClient
	Linker               linkerv1connect.LinkerServiceClient
	Weights              WeightData
	MaxDataCacheDuration time.Duration
}

type Service struct {
	db                *sql.DB
	qry               *db.Queries
	weightCourseNames []string

	Options
}

func NewService(database *sql.DB, options Options) Service {
	weightCourseNames := make([]string, len(options.Weights))
	i := 0
	for courseName := range options.Weights {
		weightCourseNames[i] = courseName
		i++
	}

	s := Service{
		db:                database,
		qry:               db.New(database),
		weightCourseNames: weightCourseNames,
		Options:           options,
	}
	go s.removeExpiredWorker(context.Background())
	go s.recacheWorker(context.Background())

	return s
}

func (s Service) removeExpiredWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ctx, span := tracer.Start(ctx, "cron_job:remove_expired_cache")
			err := s.qry.DeleteCachedStudentDataBefore(ctx, timezone.Now().Unix())
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}
			span.End()
		}
	}
}

func (s Service) recacheAllStudents(ctx context.Context) {
	studentIds, err := s.qry.GetStudents(ctx)
	if err != nil {
		return
	}

	wg := sync.WaitGroup{}
	for _, id := range studentIds {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.recacheStudentData(ctx, id)
			if err != nil {
				slog.ErrorContext(ctx, "failed to scrape student data", "err", err)
			}
		}()
	}
	wg.Wait()
}

func (s Service) recacheWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := timezone.Now()
			if now.Hour() == 7 || now.Hour() == 10 || now.Hour() == 13 || now.Hour() == 16 {
				s.recacheAllStudents(ctx)
			}
		}
	}
}

func (s Service) GetCredentialStatus(ctx context.Context, req *connect.Request[studentdatav1.GetCredentialStatusRequest]) (*connect.Response[studentdatav1.GetCredentialStatusResponse], error) {
	profile, _ := verifier.ProfileFromContext(ctx)

	psOAuthFlow, err := s.Powerschool.GetOAuthFlow(ctx, &connect.Request[powerservicev1.GetOAuthFlowRequest]{Msg: &powerservicev1.GetOAuthFlowRequest{}})
	if err != nil {
		return nil, err
	}

	psAuthStatus, err := s.Powerschool.GetAuthStatus(ctx, &connect.Request[powerservicev1.GetAuthStatusRequest]{
		Msg: &powerservicev1.GetAuthStatusRequest{
			StudentId: profile.Email,
		},
	})
	if err != nil {
		return nil, err
	}

	moodleAuthStatus, err := s.Moodle.GetAuthStatus(ctx, &connect.Request[vcsmoodlev1.GetAuthStatusRequest]{
		Msg: &vcsmoodlev1.GetAuthStatusRequest{
			StudentId: profile.Email,
		},
	})
	if err != nil {
		return nil, err
	}

	return &connect.Response[studentdatav1.GetCredentialStatusResponse]{
		Msg: &studentdatav1.GetCredentialStatusResponse{
			Statuses: []*studentdatav1.CredentialStatus{
				{
					Id:      "powerschool",
					Name:    "PowerSchool",
					Picture: "/icons/powerschool.jpg",
					LoginFlow: &studentdatav1.CredentialStatus_Oauth{
						Oauth: psOAuthFlow.Msg.GetFlow(),
					},
					Provided: psAuthStatus.Msg.GetIsAuthenticated(),
				},
				{
					Id:      "moodle",
					Name:    "Moodle",
					Picture: "/icons/moodle.jpg",
					LoginFlow: &studentdatav1.CredentialStatus_UsernamePassword{
						UsernamePassword: &studentdatav1.UsernamePasswordFlow{},
					},
					Provided: moodleAuthStatus.Msg.GetProvided(),
				},
			},
		},
	}, nil
}

func (s Service) ProvideCredential(ctx context.Context, req *connect.Request[studentdatav1.ProvideCredentialRequest]) (*connect.Response[studentdatav1.ProvideCredentialResponse], error) {
	profile, _ := verifier.ProfileFromContext(ctx)

	switch req.Msg.GetId() {
	case "powerschool":
		_, err := s.Powerschool.ProvideOAuth(ctx, &connect.Request[powerservicev1.ProvideOAuthRequest]{
			Msg: &powerservicev1.ProvideOAuthRequest{
				StudentId: profile.Email,
				Token:     req.Msg.GetOauthToken().GetToken(),
			},
		})
		if err != nil {
			return nil, err
		}
	case "moodle":
		_, err := s.Moodle.ProvideUsernamePassword(ctx, &connect.Request[vcsmoodlev1.ProvideUsernamePasswordRequest]{
			Msg: &vcsmoodlev1.ProvideUsernamePasswordRequest{
				StudentId: profile.Email,
				Username:  req.Msg.GetUsernamePassword().GetUsername(),
				Password:  req.Msg.GetUsernamePassword().GetPassword(),
			},
		})
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown credential id '%s'", req.Msg.GetId())
	}

	return &connect.Response[studentdatav1.ProvideCredentialResponse]{Msg: &studentdatav1.ProvideCredentialResponse{}}, nil
}

func (s Service) getCachedStudentData(ctx context.Context, studentEmail string) (*studentdatav1.StudentData, error) {
	ctx, span := tracer.Start(ctx, "get_cached_student_data")
	defer span.End()

	cachedRow, err := s.qry.GetCachedStudentData(ctx, studentEmail)
	if err == sql.ErrNoRows {
		err := fmt.Errorf("student data was not cached")
		span.SetStatus(codes.Ok, err.Error())
		return nil, err
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read cache")
		return nil, err
	}

	now := timezone.Now().Unix()
	if now >= cachedRow.Expiresat {
		err := fmt.Errorf("cached data has expired")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	data := &studentdatav1.StudentData{}
	err = proto.Unmarshal(cachedRow.Cached, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse cached object")
		return nil, err
	}
	return data, nil
}

func instrumentDataSnapshot(span trace.Span, message string, data *studentdatav1.StudentData) {
	span.AddEvent(
		message,
		trace.WithAttributes(attribute.String("data", protojson.Format(data))),
	)
}

var nonGradedCoursesKeywords = []string{
	"openperiod",
	"unscheduled",
	"chapel",
}

func (s Service) recacheStudentData(ctx context.Context, userEmail string) (*studentdatav1.StudentData, error) {
	ctx, span := tracer.Start(ctx, "scrape_students")
	defer span.End()

	data := &studentdatav1.StudentData{}

	psres, err := s.Powerschool.GetStudentData(ctx, &connect.Request[powerservicev1.GetStudentDataRequest]{
		Msg: &powerservicev1.GetStudentDataRequest{
			StudentId: userEmail,
		},
	})
	if err != nil {
		span.RecordError(err)
		span.AddEvent("failed to get powerschool data")
		span.SetStatus(codes.Error, "incomplete data")
	} else {
		patchStudentDataWithPowerschool(ctx, data, psres.Msg)

		instrumentDataSnapshot(span, "patched powerschool student data", data)
	}

	moodleres, err := s.Moodle.GetStudentData(ctx, &connect.Request[vcsmoodlev1.GetStudentDataRequest]{
		Msg: &vcsmoodlev1.GetStudentDataRequest{
			StudentId: userEmail,
		},
	})
	if err != nil {
		span.RecordError(err)
		span.AddEvent("failed to get moodle data")
		span.SetStatus(codes.Error, "incomplete data")
	} else {
		if psres != nil {
			linkMoodleToPowerschool(ctx, s.Linker, moodleres.Msg, psres.Msg)
		}
		patchStudentDataWithMoodle(ctx, data, moodleres.Msg)

		instrumentDataSnapshot(span, "patched moodle student data", data)
	}

	courseSnapshots := make([]*gradesnapshotsv1.PushRequest_Course, len(data.GetCourses()))
	for i, c := range data.GetCourses() {
		courseSnapshots[i] = &gradesnapshotsv1.PushRequest_Course{
			Course: c.GetName(),
			Value:  c.GetOverallGrade(),
		}
	}
	_, err = s.Gradesnapshots.Push(ctx, &connect.Request[gradesnapshotsv1.PushRequest]{
		Msg: &gradesnapshotsv1.PushRequest{
			User:    userEmail,
			Time:    timezone.Now().Unix(),
			Courses: courseSnapshots,
		},
	})
	if err != nil {
		span.RecordError(err)
		span.AddEvent("failed to push grade snapshot data")
		span.SetStatus(codes.Error, "failed to push gradesnapshot data")
	}

	gradesnapshotres, err := s.Gradesnapshots.Pull(ctx, &connect.Request[gradesnapshotsv1.PullRequest]{
		Msg: &gradesnapshotsv1.PullRequest{
			User: userEmail,
		},
	})
	if err != nil {
		span.RecordError(err)
		span.AddEvent("failed to get grade snapshot data")
		span.SetStatus(codes.Error, "incomplete data")
	} else {
		patchStudentDataWithGradeSnapshots(ctx, data, gradesnapshotres.Msg)

		instrumentDataSnapshot(span, "patched grade snapshots into student data", data)
	}

	if psres != nil {
		weights, err := linkWeightsToPowerschool(ctx, s.Linker, psres.Msg, s.Weights, s.weightCourseNames)
		if err != nil {
			span.RecordError(err)
			span.AddEvent("failed to get weight data")
			span.SetStatus(codes.Error, "incomplete data")
		} else {
			patchStudentDataWithWeights(ctx, data, weights)

			instrumentDataSnapshot(span, "patched grade weights into student data", data)
		}
	}

	newCached, err := proto.Marshal(data)
	if err != nil {
		span.RecordError(err)
		span.AddEvent("failed to marshal completed data")
		span.SetStatus(codes.Error, "failed to cache data")
	} else {
		err = s.qry.SetCachedStudentData(ctx, db.SetCachedStudentDataParams{
			Studentid: userEmail,
			Cached:    newCached,
			Expiresat: timezone.Now().Add(s.MaxDataCacheDuration).Unix(),
		})
		if err != nil {
			span.RecordError(err)
			span.AddEvent("failed to push marshaled data")
			span.SetStatus(codes.Error, "failed to cache data")
		}
	}

	return data, nil
}

func (s Service) GetStudentData(ctx context.Context, req *connect.Request[studentdatav1.GetStudentDataRequest]) (*connect.Response[studentdatav1.GetStudentDataResponse], error) {
	span := trace.SpanFromContext(ctx)

	profile, _ := verifier.ProfileFromContext(ctx)

	cachedData, err := s.getCachedStudentData(ctx, profile.Email)
	if err == nil {
		return &connect.Response[studentdatav1.GetStudentDataResponse]{
			Msg: &studentdatav1.GetStudentDataResponse{
				Data: cachedData,
			},
		}, nil
	} else {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	data, err := s.recacheStudentData(ctx, profile.Email)
	if err != nil {
		return nil, err
	}
	return &connect.Response[studentdatav1.GetStudentDataResponse]{
		Msg: &studentdatav1.GetStudentDataResponse{
			Data: data,
		},
	}, nil
}

func (s Service) RefreshData(ctx context.Context, _ *connect.Request[studentdatav1.RefreshDataRequest]) (*connect.Response[studentdatav1.RefreshDataResponse], error) {
	profile, _ := verifier.ProfileFromContext(ctx)

	data, err := s.recacheStudentData(ctx, profile.Email)
	if err != nil {
		return nil, err
	}
	return &connect.Response[studentdatav1.RefreshDataResponse]{
		Msg: &studentdatav1.RefreshDataResponse{
			Refreshed: data,
		},
	}, nil
}
