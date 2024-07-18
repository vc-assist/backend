package gradesnapshots

import (
	"context"
	"testing"
	"time"
	"vcassist-backend/lib/testutil"
	"vcassist-backend/lib/timezone"
	gradesnapshotsv1 "vcassist-backend/proto/vcassist/services/gradesnapshots/v1"
	"vcassist-backend/services/gradesnapshots/db"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	_ "embed"

	_ "modernc.org/sqlite"
)

func TestService(t *testing.T) {
	setup, cleanup := testutil.SetupService(t, testutil.ServiceParams{
		Name:     "services/gradesnapshots",
		DbSchema: db.Schema,
	})
	defer cleanup()
	service := NewService(setup.DB)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	{
		res, err := service.Pull(ctx, &connect.Request[gradesnapshotsv1.PullRequest]{
			Msg: &gradesnapshotsv1.PullRequest{
				User: "unknown-user",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		require.Len(t, res.Msg.GetCourses(), 0)
	}
	{
		_, err := service.Push(ctx, &connect.Request[gradesnapshotsv1.PushRequest]{
			Msg: &gradesnapshotsv1.PushRequest{
				User: "user",
				Time: timezone.Now().Unix(),
				Courses: []*gradesnapshotsv1.PushRequest_Course{
					{Course: "physics", Value: 24},
					{Course: "math", Value: 48},
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		_, err = service.Push(ctx, &connect.Request[gradesnapshotsv1.PushRequest]{
			Msg: &gradesnapshotsv1.PushRequest{
				User: "user",
				Time: timezone.Now().Add(time.Hour * 24).Unix(),
				Courses: []*gradesnapshotsv1.PushRequest_Course{
					{Course: "physics", Value: 27},
					{Course: "math", Value: 48},
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		_, err = service.Push(ctx, &connect.Request[gradesnapshotsv1.PushRequest]{
			Msg: &gradesnapshotsv1.PushRequest{
				User: "user1",
				Time: timezone.Now().Unix(),
				Courses: []*gradesnapshotsv1.PushRequest_Course{
					{Course: "somecourse", Value: 999},
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		res, err := service.Pull(ctx, &connect.Request[gradesnapshotsv1.PullRequest]{
			Msg: &gradesnapshotsv1.PullRequest{
				User: "user",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		require.Len(t, res.Msg.GetCourses(), 2)

		t.Log(res.Msg.GetCourses())

		var math *gradesnapshotsv1.PullResponse_Course
		var physics *gradesnapshotsv1.PullResponse_Course
		for _, c := range res.Msg.GetCourses() {
			if c.GetCourse() == "physics" {
				physics = c
			}
			if c.GetCourse() == "math" {
				math = c
			}
		}
		require.NotNil(t, physics)
		require.NotNil(t, math)
		require.Len(t, physics.GetSnapshots(), 2)
		require.Len(t, math.GetSnapshots(), 2)
	}
}
