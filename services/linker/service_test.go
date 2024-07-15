package linker

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/services/linker/api"

	_ "embed"

	_ "modernc.org/sqlite"

	"connectrpc.com/connect"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

//go:embed db/schema.sql
var schema string

func setup(t testing.TB) (Service, func()) {
	cleanup := telemetry.SetupForTesting(t, "test:linker")

	sqlite, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlite.Exec(schema)
	if err != nil {
		t.Fatal(err)
	}

	return NewService(sqlite), cleanup
}

func TestService(t *testing.T) {
	service, cleanup := setup(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	{
		res, err := service.GetKnownKeys(ctx, &connect.Request[api.GetKnownKeysRequest]{
			Msg: &api.GetKnownKeysRequest{},
		})
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, len(res.Msg.GetKeys()), 0, "expected no known keys")
	}
	{
		res, err := service.GetKnownSets(ctx, &connect.Request[api.GetKnownSetsRequest]{
			Msg: &api.GetKnownSetsRequest{},
		})
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, len(res.Msg.GetSets()), 0, "expected no known sets")
	}
	{
		res, err := service.GetExplicitLinks(ctx, &connect.Request[api.GetExplicitLinksRequest]{
			Msg: &api.GetExplicitLinksRequest{
				LeftSet:  "random set",
				RightSet: "random set 2",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, len(res.Msg.GetLeftKeys()), 0, "expected no explicit links to exist")
		require.Equal(t, len(res.Msg.GetRightKeys()), 0, "expected no explicit links to exist")
	}

	_, err := service.AddExplicitLink(ctx, &connect.Request[api.AddExplicitLinkRequest]{
		Msg: &api.AddExplicitLinkRequest{
			Left: &api.ExplicitKey{
				Set: "powerschool",
				Key: "Physics 1 (H)",
			},
			Right: &api.ExplicitKey{
				Set: "moodle",
				Key: "Physics 1",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.AddExplicitLink(ctx, &connect.Request[api.AddExplicitLinkRequest]{
		Msg: &api.AddExplicitLinkRequest{
			Left: &api.ExplicitKey{
				Set: "moodle",
				Key: "Physics 1 Honors",
			},
			Right: &api.ExplicitKey{
				Set: "powerschool",
				Key: "Physics 1 (H)",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	linkRes, err := service.Link(ctx, &connect.Request[api.LinkRequest]{
		Msg: &api.LinkRequest{
			Src: &api.Set{
				Name: "powerschool",
				Keys: []string{
					"Physics 1 (H)",
					"AP Calculus BC",
					"AP Human Geography",
				},
			},
			Dst: &api.Set{
				Name: "powerschool",
				Keys: []string{
					"Physics 1",
					"AP Calculus BC (H)",
					"AP Geography",
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(linkRes.Msg.String())
	diff := cmp.Diff(
		map[string]string{
			"Physics 1 (H)":  "Physics 1",
			"AP Calculus BC": "AP Calculus BC (H)",
		},
		linkRes.Msg.GetSrcToDst(),
	)
	if diff != "" {
		t.Fatal(diff)
	}
}
