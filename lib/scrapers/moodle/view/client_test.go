package view

import (
	"context"
	"errors"
	"sync"
	"testing"
	"vcassist-backend/lib/configutil"
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

	coreConfig, err := configutil.ReadConfig[core.TestConfig](".dev/test_moodle/config.json5")
	if err != nil {
		t.Fatal("failed to read test config at .dev/test_moodle/config.json5")
	}
	config, err := configutil.ReadConfig[TestConfig](".dev/test_moodle/view/config.json5")
	if err != nil {
		t.Fatal("there is no test config at .dev/test_moodle/view/config.json5")
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

	client, err := NewClient(ctx, coreClient)
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
		id, err := targetCourse.Id()
		if err != nil {
			t.Fatal(err)
		}
		t.Log("Target Course", targetCourse.Name, id)

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
}
