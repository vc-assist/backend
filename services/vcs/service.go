package vcs

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/protobuf/proto"
)

var tracer = otel.Tracer("services/vcs")

type Service struct {
	db             *sql.DB
	qry            *db.Queries
	gradesnapshots gradesnapshotsv1connect.GradeSnapshotsServiceClient
	powerschool    powerservicev1connect.PowerschoolServiceClient
	moodle         vcsmoodlev1connect.MoodleServiceClient
	linker         linkerv1connect.LinkerServiceClient
}

func NewService(
	database *sql.DB,
	powerschool powerservicev1connect.PowerschoolServiceClient,
	moodle vcsmoodlev1connect.MoodleServiceClient,
	linker linkerv1connect.LinkerServiceClient,
	gradesnapshots gradesnapshotsv1connect.GradeSnapshotsServiceClient,
) Service {
	return Service{
		db:             database,
		qry:            db.New(database),
		powerschool:    powerschool,
		moodle:         moodle,
		linker:         linker,
		gradesnapshots: gradesnapshots,
	}
}

func (s Service) removeExpiredWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ctx, span := tracer.Start(ctx, "removeExpiredRows")
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
	ctx, span := tracer.Start(ctx, "recacheAllStudents")
	defer span.End()

	studentIds, err := s.qry.GetStudents(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	wg := sync.WaitGroup{}
	for _, id := range studentIds {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.recacheStudentData(ctx, id)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
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

func (s Service) StartWorker(ctx context.Context) {
	go s.removeExpiredWorker(ctx)
	go s.recacheWorker(ctx)
}

func (s Service) GetCredentialStatus(ctx context.Context, req *connect.Request[studentdatav1.GetCredentialStatusRequest]) (*connect.Response[studentdatav1.GetCredentialStatusResponse], error) {
	ctx, span := tracer.Start(ctx, "GetCredentialStatus")
	defer span.End()

	profile, _ := verifier.ProfileFromContext(ctx)

	psOAuthFlow, err := s.powerschool.GetOAuthFlow(ctx, &connect.Request[powerservicev1.GetOAuthFlowRequest]{Msg: &powerservicev1.GetOAuthFlowRequest{}})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	psAuthStatus, err := s.powerschool.GetAuthStatus(ctx, &connect.Request[powerservicev1.GetAuthStatusRequest]{
		Msg: &powerservicev1.GetAuthStatusRequest{
			StudentId: profile.Email,
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	moodleAuthStatus, err := s.moodle.GetAuthStatus(ctx, &connect.Request[vcsmoodlev1.GetAuthStatusRequest]{
		Msg: &vcsmoodlev1.GetAuthStatusRequest{
			StudentId: profile.Email,
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
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
	ctx, span := tracer.Start(ctx, "ProvideCredential")
	defer span.End()

	profile, _ := verifier.ProfileFromContext(ctx)

	switch req.Msg.GetId() {
	case "powerschool":
		_, err := s.powerschool.ProvideOAuth(ctx, &connect.Request[powerservicev1.ProvideOAuthRequest]{
			Msg: &powerservicev1.ProvideOAuthRequest{
				StudentId: profile.Email,
				Token:     req.Msg.GetOauthToken().GetToken(),
			},
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	case "moodle":
		_, err := s.moodle.ProvideUsernamePassword(ctx, &connect.Request[vcsmoodlev1.ProvideUsernamePasswordRequest]{
			Msg: &vcsmoodlev1.ProvideUsernamePasswordRequest{
				StudentId: req.Msg.GetId(),
				Username:  req.Msg.GetUsernamePassword().GetUsername(),
				Password:  req.Msg.GetUsernamePassword().GetPassword(),
			},
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}

	return &connect.Response[studentdatav1.ProvideCredentialResponse]{Msg: &studentdatav1.ProvideCredentialResponse{}}, nil
}

func (s Service) getCachedStudentData(ctx context.Context, studentEmail string) (*studentdatav1.StudentData, error) {
	ctx, span := tracer.Start(ctx, "getCachedStudentData")
	defer span.End()

	cachedRow, err := s.qry.GetCachedStudentData(ctx, studentEmail)
	if err == nil {
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

	var data *studentdatav1.StudentData
	err = proto.Unmarshal(cachedRow.Cached, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse cached object")
		return nil, err
	}
	return data, nil
}

func (s Service) recacheStudentData(ctx context.Context, userEmail string) (*studentdatav1.StudentData, error) {
	ctx, span := tracer.Start(ctx, "recacheStudentData")
	defer span.End()

	data := &studentdatav1.StudentData{}

	psres, err := s.powerschool.GetStudentData(ctx, &connect.Request[powerservicev1.GetStudentDataRequest]{
		Msg: &powerservicev1.GetStudentDataRequest{
			StudentId: userEmail,
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get powerschool data")
	} else {
		patchStudentDataWithPowerschool(ctx, data, psres.Msg)
	}

	moodleres, err := s.moodle.GetStudentData(ctx, &connect.Request[vcsmoodlev1.GetStudentDataRequest]{
		Msg: &vcsmoodlev1.GetStudentDataRequest{
			StudentId: userEmail,
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get moodle data")
	} else {
		linkMoodleToPowerschool(ctx, s.linker, moodleres.Msg, psres.Msg)
		patchStudentDataWithMoodle(ctx, data, moodleres.Msg)
	}

	now := timezone.Now()
	snapshots := make([]*gradesnapshotsv1.CourseSnapshot, len(data.GetCourses()))
	for i, c := range data.GetCourses() {
		snapshots[i] = &gradesnapshotsv1.CourseSnapshot{
			Course: c.GetName(),
			Snapshot: &gradesnapshotsv1.Snapshot{
				Value: c.GetOverallGrade(),
				Time:  now.Unix(),
			},
		}
	}
	_, err = s.gradesnapshots.Push(ctx, &connect.Request[gradesnapshotsv1.PushRequest]{
		Msg: &gradesnapshotsv1.PushRequest{
			Snapshot: &gradesnapshotsv1.UserSnapshot{
				User:    userEmail,
				Courses: snapshots,
			},
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to push grade snapshots")
	}

	gradesnapshotres, err := s.gradesnapshots.Pull(ctx, &connect.Request[gradesnapshotsv1.PullRequest]{
		Msg: &gradesnapshotsv1.PullRequest{
			User: userEmail,
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get grade snapshot data")
	} else {
		patchStudentDataWithGradeSnapshots(ctx, data, gradesnapshotres.Msg)
	}

	weights, err := getWeightsForPowerschool(ctx, s.linker, psres.Msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get weight data")
	} else {
		patchStudentDataWithWeights(ctx, data, weights)
	}

	newCached, err := proto.Marshal(data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	err = s.qry.SetCachedStudentData(ctx, db.SetCachedStudentDataParams{
		Studentid: userEmail,
		Cached:    newCached,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return data, nil
}

func (s Service) GetStudentData(ctx context.Context, req *connect.Request[studentdatav1.GetStudentDataRequest]) (*connect.Response[studentdatav1.GetStudentDataResponse], error) {
	ctx, span := tracer.Start(ctx, "GetStudentData")
	defer span.End()

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
	ctx, span := tracer.Start(ctx, "RefreshData")
	defer span.End()

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
