package keychain

import (
	"context"
	"testing"
	"time"
	"vcassist-backend/lib/testutil"
	keychainv1 "vcassist-backend/proto/vcassist/services/keychain/v1"
	"vcassist-backend/services/keychain/db"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
)

func TestService(t *testing.T) {
	res, cleanup := testutil.SetupService(t, testutil.ServiceParams{
		Name:     "services/keychain",
		DbSchema: db.Schema,
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	service := NewService(ctx, res.DB)

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
