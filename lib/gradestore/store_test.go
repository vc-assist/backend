package gradestore

import (
	"context"
	"testing"
	"time"
	"vcassist-backend/lib/gradestore/db"
	"vcassist-backend/lib/testutil"
	"vcassist-backend/lib/timezone"

	"github.com/stretchr/testify/require"

	_ "embed"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

func TestStore(t *testing.T) {
	setup, cleanup := testutil.SetupService(t, testutil.ServiceParams{
		Name:     "test/gradestore",
		DbSchema: db.Schema,
	})
	defer cleanup()
	store := NewStore(setup.DB)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	{
		res, err := store.Pull(ctx, "unknown-user")
		if err != nil {
			t.Fatal(err)
		}
		require.Len(t, res, 0)
	}
	{
		err := store.Push(ctx, PushRequest{
			Time: timezone.Now(),
			Users: []UserSnapshot{
				{
					User: "alice",
					Courses: []CourseSnapshot{
						{Course: "physics", Value: 24},
						{Course: "math", Value: 48},
					},
				},
				{
					User: "bob",
					Courses: []CourseSnapshot{
						{Course: "chemistry", Value: 38},
					},
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		err = store.Push(ctx, PushRequest{
			Time: timezone.Now().Add(time.Hour * 24),
			Users: []UserSnapshot{
				{
					User: "alice",
					Courses: []CourseSnapshot{
						{Course: "physics", Value: 27},
						{Course: "math", Value: 48},
					},
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		err = store.Push(ctx, PushRequest{
			Time: timezone.Now(),
			Users: []UserSnapshot{
				{
					User: "user1",
					Courses: []CourseSnapshot{
						{Course: "somecourse", Value: 999},
					},
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		res, err := store.Pull(ctx, "alice")
		if err != nil {
			t.Fatal(err)
		}
		require.Len(t, res, 2)

		t.Log(res)

		var math CourseSnapshotSeries
		var physics CourseSnapshotSeries
		for _, c := range res {
			if c.Course == "physics" {
				physics = c
			}
			if c.Course == "math" {
				math = c
			}
		}
		require.NotNil(t, physics)
		require.NotNil(t, math)
		require.Len(t, physics.Snapshots, 2)
		require.Len(t, math.Snapshots, 2)
	}
}
