package view

import (
	"context"
	"errors"
	"sync"
	"testing"
	devenv "vcassist-backend/dev/env"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/telemetry"

	"github.com/stretchr/testify/require"
)

type TestConfig struct {
	TargetCourse string `json:"target_course"`
}

func TestClient(t *testing.T) {
	cleanup := telemetry.SetupForTesting("test:scrapers/moodle/view")
	defer cleanup()

	ctx, span := tracer.Start(context.Background(), "TestClient")
	defer span.End()

	coreConfig, err := devenv.GetStateConfig[core.TestConfig]("moodle/core.json5")
	if err != nil {
		t.Skip("skipping because failed to read test config at .dev/state/moodle/core.json5")
	}
	config, err := devenv.GetStateConfig[TestConfig]("moodle/view.json5")
	if err != nil {
		t.Skip("skipping because there is no test config at .dev/state/moodle/view.json5")
	}

	coreClient, err := core.NewClient(ctx, core.ClientOptions{
		BaseUrl: coreConfig.BaseUrl,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = coreClient.LoginUsernamePassword(ctx, coreConfig.Username, coreConfig.Password)
	if err != nil {
		t.Fatal(err)
	}

	client, err := NewClient(ctx, coreClient, ClientOptions{
		ClientId: coreConfig.Username,
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
			if c.Name == config.TargetCourse {
				targetCourse = c
				break
			}
		}
	})

	if targetCourse == (Course{}) {
		t.Fatal("could not find target course", config.TargetCourse)
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
