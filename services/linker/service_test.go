package linker

import (
	"context"
	"fmt"
	"testing"
	"time"
	"vcassist-backend/lib/testutil"
	linkerv1 "vcassist-backend/proto/vcassist/services/linker/v1"
	"vcassist-backend/services/linker/db"

	"connectrpc.com/connect"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestService(t *testing.T) {
	res, cleanup := testutil.SetupService(t, testutil.ServiceParams{
		Name:     "services/linker",
		DbSchema: db.Schema,
	})
	defer cleanup()
	service := NewService(res.DB)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	{
		res, err := service.GetKnownKeys(ctx, &connect.Request[linkerv1.GetKnownKeysRequest]{
			Msg: &linkerv1.GetKnownKeysRequest{},
		})
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, len(res.Msg.GetKeys()), 0, "expected no known keys")
	}
	{
		res, err := service.GetKnownSets(ctx, &connect.Request[linkerv1.GetKnownSetsRequest]{
			Msg: &linkerv1.GetKnownSetsRequest{},
		})
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, len(res.Msg.GetSets()), 0, "expected no known sets")
	}
	{
		res, err := service.GetExplicitLinks(ctx, &connect.Request[linkerv1.GetExplicitLinksRequest]{
			Msg: &linkerv1.GetExplicitLinksRequest{
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

	_, err := service.AddExplicitLink(ctx, &connect.Request[linkerv1.AddExplicitLinkRequest]{
		Msg: &linkerv1.AddExplicitLinkRequest{
			Left: &linkerv1.ExplicitKey{
				Set: "powerschool",
				Key: "Physics 1 (H)",
			},
			Right: &linkerv1.ExplicitKey{
				Set: "moodle",
				Key: "Physics 1",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.AddExplicitLink(ctx, &connect.Request[linkerv1.AddExplicitLinkRequest]{
		Msg: &linkerv1.AddExplicitLinkRequest{
			Left: &linkerv1.ExplicitKey{
				Set: "moodle",
				Key: "Physics 1 Honors",
			},
			Right: &linkerv1.ExplicitKey{
				Set: "powerschool",
				Key: "Physics 1 (H)",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	linkRes, err := service.Link(ctx, &connect.Request[linkerv1.LinkRequest]{
		Msg: &linkerv1.LinkRequest{
			Src: &linkerv1.Set{
				Name: "powerschool",
				Keys: []string{
					"Physics 1 (H)",
					"AP Calculus BC",
					"AP Human Geography",
				},
			},
			Dst: &linkerv1.Set{
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
