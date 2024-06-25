package student

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	devenv "vcassist-backend/dev/setup"
	"vcassist-backend/lib/htmlutil"

	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"
)

func getTestConfig(t testing.TB) devenv.MoodleTestConfig {
	contents, err := devenv.GetStateFile("moodle_credentials.json")
	if err != nil {
		t.Fatal(err)
	}

	var cached devenv.MoodleTestConfig
	err = json.Unmarshal(contents, &cached)
	if err != nil {
		t.Fatal(err)
	}
	return cached
}

func TestClient(t *testing.T) {
	ctx, span := tracer.Start(context.Background(), "TestClient")
	defer span.End()

	config := getTestConfig(t)
	cache, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewClient(ctx, ClientOptions{
		Cache:    cache,
		ClientId: config.Username,
		BaseUrl:  config.BaseUrl,
		Username: config.Username,
		Password: config.Password,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = client.LoginUsernamePassword(ctx, config.Username, config.Password)
	if err != nil {
		t.Fatal(err)
	}

	courses, err := client.Courses(ctx)
	if err != nil {
		t.Fatal(err)
	}
	require.Greater(t, len(courses), 0)
	require.True(t, slices.ContainsFunc(courses, func(e htmlutil.Anchor) bool {
		return e.Name == "VC Assist"
	}))

	t.Log("Courses", courses)

	sections, err := client.Sections(ctx, courses[0])
	if err != nil {
		t.Fatal(err)
	}
	require.Greater(t, len(sections), 0)

	t.Log("Resources", sections)

	resources, err := client.Resources(ctx, sections[0])
	if err != nil {
		t.Fatal(err)
	}
	require.Greater(t, len(resources), 0)

	t.Log("Resources", resources)

	chapters, err := client.Chapters(ctx, resources[0])
	if err != nil {
		t.Fatal(err)
	}
	require.Greater(t, len(chapters), 0)

	t.Log("Chapters", chapters)
}
