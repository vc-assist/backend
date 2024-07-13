package view

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	devenv "vcassist-backend/dev/env"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/telemetry"

	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"
)

func getTestConfig(t testing.TB) devenv.MoodleTestConfig {
	contents, err := devenv.GetStateFile("moodle_config.json")
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
	cleanup := telemetry.SetupForTesting(t, "test:moodle/view")
	defer cleanup()

	ctx, span := tracer.Start(context.Background(), "TestClient")
	defer span.End()

	config := getTestConfig(t)
	cache, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		t.Fatal(err)
	}

	coreClient, err := core.NewClient(ctx, core.ClientOptions{
		BaseUrl: config.BaseUrl,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = coreClient.LoginUsernamePassword(ctx, config.Username, config.Password)
	if err != nil {
		t.Fatal(err)
	}

	client, err := NewClient(ctx, coreClient, ClientOptions{
		Cache:    cache,
		ClientId: config.Username,
	})
	if err != nil {
		t.Fatal(err)
	}

	var targetCourse Course
	t.Run("TestCourses", func(t *testing.T) {
		courses, err := client.Courses(ctx)
		if err != nil {
			t.Fatal(err)
		}
		require.Greater(t, len(courses), 0, "could not find any courses on this moodle account")

		t.Log("Courses", courses)

		for _, c := range courses {
			if c == (Course{}) {
				t.Fatal("got empty course in course list")
			}
			if c.Name == config.ViewConfig.TargetCourse {
				targetCourse = c
				break
			}
		}
	})

	if targetCourse == (Course{}) {
		t.Fatal("could not find target course", config.ViewConfig.TargetCourse)
	}

	t.Run("TestSections", func(t *testing.T) {
		t.Log("Target Course", targetCourse.Name, targetCourse.Id())

		sections, err := client.Sections(ctx, targetCourse)
		if err != nil {
			t.Fatal(err)
		}
		require.Greater(t, len(sections), 0, "could not find any sections in the target course", targetCourse.Name)

		t.Log("Resources", sections)

		var errs []error
		errLock := sync.Mutex{}
		hasResources := false
		wg := sync.WaitGroup{}

		for _, s := range sections {
			wg.Add(1)
			go func(s Section) {
				defer wg.Done()

				resources, err := client.Resources(ctx, s)
				if err != nil {
					errLock.Lock()
					defer errLock.Unlock()
					errs = append(errs, err)
					return
				}

				if len(resources) > 0 {
					hasResources = true
				}
				t.Log("Resources", resources)
			}(s)
		}

		wg.Wait()

		if len(errs) > 0 {
			t.Fatal(errors.Join(errs...))
		}
		if !hasResources {
			t.Fatal("no section has at least one resource, this may be a bug or the course in question may just not have any resources.")
		}
	})

	t.Run("TestChapters", func(t *testing.T) {
		t.Skip("currently unimplemented")
	})
}
