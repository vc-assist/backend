package vcsmoodle

import (
	"context"
	"testing"
	"time"
	devenv "vcassist-backend/dev/env"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/testutil"
	vcsmoodlev1 "vcassist-backend/proto/vcassist/services/vcsmoodle/v1"
	"vcassist-backend/proto/vcassist/services/vcsmoodle/v1/vcsmoodlev1connect"
	"vcassist-backend/services/keychain"
	keychaindb "vcassist-backend/services/keychain/db"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
)

func setup(t testing.TB) (vcsmoodlev1connect.MoodleServiceClient, func()) {
	keyRes, cleanupKeychain := testutil.SetupService(t, testutil.ServiceParams{
		Name:     "services/keychain",
		DbSchema: keychaindb.Schema,
	})
	_, cleanupSelf := testutil.SetupService(t, testutil.ServiceParams{
		Name: "services/vcsmoodle",
	})

	ctx, cancelKeychain := context.WithCancel(context.Background())
	keychainService := keychain.NewService(ctx, keyRes.DB)
	s := NewService(keychainService)

	return s, func() {
		cleanupKeychain()
		cleanupSelf()
		cancelKeychain()
	}
}

func TestService(t *testing.T) {
	service, cleanup := setup(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	coreConfig, err := devenv.GetStateConfig[core.TestConfig]("vcsmoodle.json5")
	if err != nil {
		t.Skip("skipping test because no valid test config was found at dev/.state/vcsmoodle.json5")
	}

	{
		res, err := service.GetAuthStatus(ctx, &connect.Request[vcsmoodlev1.GetAuthStatusRequest]{
			Msg: &vcsmoodlev1.GetAuthStatusRequest{
				StudentId: "unknown-id",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		require.False(t, res.Msg.GetProvided())
	}

	studentId := "test-student"
	{
		_, err = service.ProvideUsernamePassword(ctx, &connect.Request[vcsmoodlev1.ProvideUsernamePasswordRequest]{
			Msg: &vcsmoodlev1.ProvideUsernamePasswordRequest{
				StudentId: studentId,
				Username:  coreConfig.Username,
				Password:  coreConfig.Password,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		res, err := service.GetAuthStatus(ctx, &connect.Request[vcsmoodlev1.GetAuthStatusRequest]{
			Msg: &vcsmoodlev1.GetAuthStatusRequest{
				StudentId: studentId,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		require.True(t, res.Msg.GetProvided())
	}
	{
		res, err := service.GetStudentData(ctx, &connect.Request[vcsmoodlev1.GetStudentDataRequest]{
			Msg: &vcsmoodlev1.GetStudentDataRequest{
				StudentId: studentId,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		require.Greater(t, len(res.Msg.GetCourses()), 0)

		for _, c := range res.Msg.GetCourses() {
			require.NotEmpty(t, c.GetName())
			lessonPlan := c.GetLessonPlan()
			zoomLink := c.GetZoomLink()
			if lessonPlan == "" {
				t.Log("WARN", c.GetName(), "has an empty lesson plan")
			}
			if zoomLink == "" {
				t.Log("WARN", c.GetName(), "has an empty zoom link")
			}
		}
	}
}
