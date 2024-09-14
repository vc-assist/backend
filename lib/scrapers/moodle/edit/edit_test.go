package edit

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"vcassist-backend/lib/configutil"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/scrapers/moodle/view"
	"vcassist-backend/lib/telemetry"

	"github.com/stretchr/testify/require"
)

func setupClients(t testing.TB, ctx context.Context, config core.TestConfig) (*core.Client, view.Client) {
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

	client, err := view.NewClient(ctx, coreClient)
	if err != nil {
		t.Fatal(err)
	}

	return coreClient, client
}

type TestConfig struct {
	TargetCourse string `json:"target_course"`
}

func setup(t testing.TB, ctx context.Context) Course {
	coreConfig, err := configutil.ReadConfig[core.TestConfig](".dev/test_moodle/config.json5")
	if err != nil {
		t.Fatal("failed to read test config at .dev/test_moodle/config.json5")
	}
	config, err := configutil.ReadConfig[TestConfig](".dev/test_moodle/edit/config.json5")
	if err != nil {
		t.Fatal("there is no test config at .dev/test_moodle/edit/config.json5")
	}
	coreClient, viewClient := setupClients(t, ctx, coreConfig)

	courses, err := viewClient.Courses(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("courses", courses)

	for _, c := range courses {
		if c.Name == config.TargetCourse {
			id, err := c.Id()
			if err != nil {
				t.Fatal(err)
			}
			course, err := NewCourse(ctx, int(id), coreClient)
			if err != nil {
				t.Fatal(err)
			}
			return course
		}
	}

	t.Fatalf("failed to find specified target course '%s'", config.TargetCourse)
	return Course{}
}

func TestSections(t *testing.T) {
	cleanup := telemetry.SetupForTesting("test:scrapers/moodle/edit")
	defer cleanup()

	ctx, span := tracer.Start(context.Background(), "TestCourse")
	defer span.End()

	course := setup(t, ctx)

	var err error
	var originalSections []Section

	originalSections, err = course.ListSections(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("initial sections", originalSections)
	require.Greater(t, len(originalSections), 0, "course has no sections")
	for _, s := range originalSections {
		require.NotEmpty(t, s.Id)
		require.NotEmpty(t, s.Name)
	}

	afterCreate, err := course.CreateSections(ctx, originalSections[len(originalSections)-2].Id, 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range afterCreate {
		require.NotEmpty(t, s.Id)
		require.NotEmpty(t, s.Name)
	}
	t.Log("sections after creation", afterCreate)
	if len(afterCreate)-len(originalSections) != 2 {
		t.Fatal(
			"new sections count is not 2 more than the original sections",
			afterCreate, originalSections,
		)
	}

	var addedSectionIds []string
newSections:
	for _, section := range afterCreate {
		for _, comp := range originalSections {
			if comp.Id == section.Id {
				continue newSections
			}
		}
		addedSectionIds = append(addedSectionIds, section.Id)
	}

	t.Log("added section ids", addedSectionIds)

	expectedNames := make([]string, len(addedSectionIds))
	renameEntries := make([]RenameEntry, len(addedSectionIds))
	for i, added := range addedSectionIds {
		name := fmt.Sprintf("Renamed %d", rand.Int())
		expectedNames[i] = name
		renameEntries[i] = RenameEntry{SectionId: added, NewName: name}
	}
	err = course.RenameSections(ctx, renameEntries)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("expected names", expectedNames)

	afterRename, err := course.ListSections(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range afterRename {
		require.NotEmpty(t, s.Id)
		require.NotEmpty(t, s.Name)
	}
expected:
	for _, expected := range expectedNames {
		for _, s := range afterRename {
			if s.Name == expected {
				continue expected
			}
		}
		t.Fatalf(
			"could not find renamed section '%s' in ListSections %v",
			expected, afterRename,
		)
	}

	err = course.DeleteSections(ctx, addedSectionIds)
	if err != nil {
		t.Fatal(err)
	}
	afterDelete, err := course.ListSections(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range afterDelete {
		require.NotEmpty(t, s.Id)
		require.NotEmpty(t, s.Name)
	}

	t.Log("sections after delete", afterDelete)

	for _, added := range addedSectionIds {
		for _, s := range afterDelete {
			if s.Id == added {
				t.Fatalf(
					"should have deleted added section with id '%s' instead found it in the listed sections %v",
					added, originalSections,
				)
			}
		}
	}
}
