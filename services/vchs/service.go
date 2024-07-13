package vchs

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
	"vcassist-backend/lib/timezone"
	"vcassist-backend/services/auth/verifier"
	gradesnapshotpb "vcassist-backend/services/gradesnapshots/api"
	gradesnapshotrpc "vcassist-backend/services/gradesnapshots/api/apiconnect"
	linkerrpc "vcassist-backend/services/linker/api/apiconnect"
	pspb "vcassist-backend/services/powerschool/api"
	psrpc "vcassist-backend/services/powerschool/api/apiconnect"
	"vcassist-backend/services/studentdata/api"
	"vcassist-backend/services/vchs/db"
	moodlepb "vcassist-backend/services/vchsmoodle/api"
	moodlerpc "vcassist-backend/services/vchsmoodle/api/apiconnect"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/protobuf/proto"
)

var tracer = otel.Tracer("services/vchs")

type Service struct {
	db             *sql.DB
	qry            *db.Queries
	gradesnapshots gradesnapshotrpc.GradeSnapshotsServiceClient
	powerschool    psrpc.PowerschoolServiceClient
	moodle         moodlerpc.MoodleServiceClient
	linker         linkerrpc.LinkerServiceClient
}

func NewService(
	database *sql.DB,
	powerschool psrpc.PowerschoolServiceClient,
	moodle moodlerpc.MoodleServiceClient,
	linker linkerrpc.LinkerServiceClient,
) Service {
	return Service{
		db:          database,
		qry:         db.New(database),
		powerschool: powerschool,
		moodle:      moodle,
		linker:      linker,
	}
}

func (s Service) removeExpiredWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
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
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
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

func (s Service) GetCredentialStatus(ctx context.Context, req *connect.Request[api.GetCredentialStatusRequest]) (*connect.Response[api.GetCredentialStatusResponse], error) {
	ctx, span := tracer.Start(ctx, "GetCredentialStatus")
	defer span.End()

	profile, _ := verifier.ProfileFromContext(ctx)

	psoauthflow, err := s.powerschool.GetOAuthFlow(ctx, &connect.Request[pspb.GetOAuthFlowRequest]{Msg: &pspb.GetOAuthFlowRequest{}})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	psauthstatus, err := s.powerschool.GetAuthStatus(ctx, &connect.Request[pspb.GetAuthStatusRequest]{
		Msg: &pspb.GetAuthStatusRequest{
			StudentId: profile.Email,
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return &connect.Response[api.GetCredentialStatusResponse]{
		Msg: &api.GetCredentialStatusResponse{
			Statuses: []*api.CredentialStatus{
				{
					Id:   "powerschool",
					Name: "PowerSchool",
					LoginFlow: &api.CredentialStatus_Oauth{
						Oauth: psoauthflow.Msg.Flow,
					},
					Provided: psauthstatus.Msg.IsAuthenticated,
				},
			},
		},
	}, nil
}

func (s Service) ProvideCredential(ctx context.Context, req *connect.Request[api.ProvideCredentialRequest]) (*connect.Response[api.ProvideCredentialResponse], error) {
	ctx, span := tracer.Start(ctx, "ProvideCredential")
	defer span.End()

	profile, _ := verifier.ProfileFromContext(ctx)

	switch req.Msg.Id {
	case "powerschool":
		_, err := s.powerschool.ProvideOAuth(ctx, &connect.Request[pspb.ProvideOAuthRequest]{
			Msg: &pspb.ProvideOAuthRequest{
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
		_, err := s.moodle.ProvideUsernamePassword(ctx, &connect.Request[moodlepb.ProvideUsernamePasswordRequest]{
			Msg: &moodlepb.ProvideUsernamePasswordRequest{
				StudentId: req.Msg.Id,
				Username:  req.Msg.GetUsernamePassword().Username,
				Password:  req.Msg.GetUsernamePassword().Password,
			},
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}

	return &connect.Response[api.ProvideCredentialResponse]{Msg: &api.ProvideCredentialResponse{}}, nil
}

func (s Service) getCachedStudentData(ctx context.Context, studentEmail string) (*api.StudentData, error) {
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

	var data *api.StudentData
	err = proto.Unmarshal(cachedRow.Cached, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse cached object")
		return nil, err
	}
	return data, nil
}

func (s Service) recacheStudentData(ctx context.Context, userEmail string) (*api.StudentData, error) {
	ctx, span := tracer.Start(ctx, "recacheStudentData")
	defer span.End()

	data := &api.StudentData{}

	psres, err := s.powerschool.GetStudentData(ctx, &connect.Request[pspb.GetStudentDataRequest]{
		Msg: &pspb.GetStudentDataRequest{
			StudentId: userEmail,
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get powerschool data")
	} else {
		patchStudentDataWithPowerschool(ctx, data, psres.Msg)
	}

	moodleres, err := s.moodle.GetStudentData(ctx, &connect.Request[moodlepb.GetStudentDataRequest]{
		Msg: &moodlepb.GetStudentDataRequest{
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
	snapshots := make([]*gradesnapshotpb.CourseSnapshot, len(data.Courses))
	for i, c := range data.Courses {
		snapshots[i] = &gradesnapshotpb.CourseSnapshot{
			Course: c.Name,
			Snapshot: &gradesnapshotpb.Snapshot{
				Value: c.OverallGrade,
				Time:  now.Unix(),
			},
		}
	}
	_, err = s.gradesnapshots.Push(ctx, &connect.Request[gradesnapshotpb.PushRequest]{
		Msg: &gradesnapshotpb.PushRequest{
			Snapshot: &gradesnapshotpb.UserSnapshot{
				User:    userEmail,
				Courses: snapshots,
			},
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to push grade snapshots")
	}

	gradesnapshotres, err := s.gradesnapshots.Pull(ctx, &connect.Request[gradesnapshotpb.PullRequest]{
		Msg: &gradesnapshotpb.PullRequest{
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

func (s Service) GetStudentData(ctx context.Context, req *connect.Request[api.GetStudentDataRequest]) (*connect.Response[api.GetStudentDataResponse], error) {
	ctx, span := tracer.Start(ctx, "GetStudentData")
	defer span.End()

	profile, _ := verifier.ProfileFromContext(ctx)

	cachedData, err := s.getCachedStudentData(ctx, profile.Email)
	if err == nil {
		return &connect.Response[api.GetStudentDataResponse]{
			Msg: &api.GetStudentDataResponse{
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
	return &connect.Response[api.GetStudentDataResponse]{
		Msg: &api.GetStudentDataResponse{
			Data: data,
		},
	}, nil
}

func (s Service) RefreshData(ctx context.Context, _ *connect.Request[api.RefreshDataRequest]) (*connect.Response[api.RefreshDataResponse], error) {
	ctx, span := tracer.Start(ctx, "RefreshData")
	defer span.End()

	profile, _ := verifier.ProfileFromContext(ctx)

	data, err := s.recacheStudentData(ctx, profile.Email)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.RefreshDataResponse]{
		Msg: &api.RefreshDataResponse{
			Refreshed: data,
		},
	}, nil
}
