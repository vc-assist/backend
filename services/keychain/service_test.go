package keychain

import (
	"context"
	"database/sql"
	"testing"
	"time"
	"vcassist-backend/lib/telemetry"
	keychainv1 "vcassist-backend/proto/vcassist/services/keychain/v1"

	_ "embed"

	_ "modernc.org/sqlite"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
)

//go:embed db/schema.sql
var schema string

func setup(t testing.TB) (Service, func()) {
	cleanup := telemetry.SetupForTesting(t, "test:services/keychain")

	sqlite, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlite.Exec(schema)
	if err != nil {
		t.Fatal(err)
	}

	s := NewService(sqlite)
	return s, cleanup
}

func TestService(t *testing.T) {
	service, cleanup := setup(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	service.StartOAuthDaemon(ctx)

	{
		res, err := service.GetOAuth(ctx, &connect.Request[keychainv1.GetOAuthRequest]{
			Msg: &keychainv1.GetOAuthRequest{
				Namespace: "powerschool",
				Id:        "unknown-id",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		require.Nil(t, res.Msg.GetKey())
	}
	{
		res, err := service.GetUsernamePassword(ctx, &connect.Request[keychainv1.GetUsernamePasswordRequest]{
			Msg: &keychainv1.GetUsernamePasswordRequest{
				Namespace: "powerschool",
				Id:        "unknown-id",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		require.Nil(t, res.Msg.GetKey())
	}

	{
		_, err := service.SetUsernamePassword(ctx, &connect.Request[keychainv1.SetUsernamePasswordRequest]{
			Msg: &keychainv1.SetUsernamePasswordRequest{
				Namespace: "powerschool",
				Id:        "alice",
				Key: &keychainv1.UsernamePasswordKey{
					Username: "alice_user",
					Password: "alice_pass",
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		_, err := service.SetUsernamePassword(ctx, &connect.Request[keychainv1.SetUsernamePasswordRequest]{
			Msg: &keychainv1.SetUsernamePasswordRequest{
				Namespace: "powerschool",
				Id:        "bob",
				Key: &keychainv1.UsernamePasswordKey{
					Username: "bob_user",
					Password: "bob_pass",
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		res, err := service.GetUsernamePassword(ctx, &connect.Request[keychainv1.GetUsernamePasswordRequest]{
			Msg: &keychainv1.GetUsernamePasswordRequest{
				Namespace: "powerschool",
				Id:        "alice",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, "alice_user", res.Msg.GetKey().GetUsername())
		require.Equal(t, "alice_pass", res.Msg.GetKey().GetPassword())
	}
	{
		_, err := service.SetOAuth(ctx, &connect.Request[keychainv1.SetOAuthRequest]{
			Msg: &keychainv1.SetOAuthRequest{
				Namespace: "moodle",
				Id:        "bob",
				Key: &keychainv1.OAuthKey{
					Token:      "moodle_token",
					RefreshUrl: "https://example.url/refresh_url",
					ClientId:   "client_id",
					ExpiresAt:  time.Now().Add(time.Hour * 24).Unix(),
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		res, err := service.GetOAuth(ctx, &connect.Request[keychainv1.GetOAuthRequest]{
			Msg: &keychainv1.GetOAuthRequest{
				Namespace: "moodle",
				Id:        "bob",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, "moodle_token", res.Msg.GetKey().GetToken())
		require.Equal(t, "https://example.url/refresh_url", res.Msg.GetKey().GetRefreshUrl())
		require.Equal(t, "client_id", res.Msg.GetKey().GetClientId())
	}
	{
		res, err := service.GetUsernamePassword(ctx, &connect.Request[keychainv1.GetUsernamePasswordRequest]{
			Msg: &keychainv1.GetUsernamePasswordRequest{
				Namespace: "powerschool",
				Id:        "bob",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, "bob_user", res.Msg.GetKey().GetUsername())
		require.Equal(t, "bob_pass", res.Msg.GetKey().GetPassword())
	}
}
