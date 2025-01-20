package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"vcassist-backend/internal/components/chrono"
	"vcassist-backend/internal/components/db"
	"vcassist-backend/internal/scrapers/moodle"
	testutil "vcassist-backend/test/util"

	"github.com/stretchr/testify/require"
)

func IntegrationTestMoodle(t *testing.T) {
	username := testutil.MustFindEnv(t, "TEST_MOODLE_USERNAME")
	password := testutil.MustFindEnv(t, "TEST_MOODLE_PASSWORD")
	dbtx := testutil.OpenInMemoryDB(t)
	qry := db.New(dbtx)

	tel.ReportDebug("testing moodle scraping")

	scraper := moodle.NewScraper(
		qry,
		db.NewMakeTx(dbtx),
		chrono.NewStandardTime(),
		tel,
		username,
		password,
	)

	ctx := context.Background()

	accountId, err := qry.AddMoodleAccount(ctx, db.AddMoodleAccountParams{
		Username: username,
		Password: password,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = scraper.ScrapeUser(ctx, accountId)
	if err != nil {
		t.Fatal(err)
	}
	courseIds, err := scraper.QueryUserCourseIds(ctx, accountId)
	if err != nil {
		t.Fatal(err)
	}

	tel.ReportDebug("course ids", courseIds)
	require.NotEmpty(t, courseIds)

	err = scraper.ScrapeAll(ctx)
	if err != nil {
		t.Fatal(err)
	}

	res, err := scraper.QueryLessonPlans(ctx, courseIds)
	if err != nil {
		t.Fatal(err)
	}

	require.NotEmpty(t, res.Courses)
	for _, course := range res.Courses {
		require.GreaterOrEqual(t, course.GetId(), 0)
		require.NotEmpty(t, course.GetName())
		require.NotEmpty(t, course.GetTeacher())
		require.NotEmpty(t, course.GetUrl())
		require.NotEmpty(t, course.GetChapters())

		tel.ReportDebug(
			"course",
			course.GetId(),
			course.GetName(),
			course.GetTeacher(),
			course.GetUrl(),
		)

		foundLessonPlan := false
		for _, chapter := range course.GetChapters() {
			require.GreaterOrEqual(t, chapter.GetId(), 0)
			require.NotEmpty(t, chapter.GetUrl())
			require.NotEmpty(t, chapter.GetDates())

			var dates strings.Builder
			for i, d := range chapter.GetDates() {
				if i != 0 {
					dates.WriteString(", ")
				}
				dates.WriteString(fmt.Sprintf(
					"%d/%d",
					d.AsTime().Month(),
					d.AsTime().Day(),
				))
			}

			tel.ReportDebug("lesson plan", chapter.GetId(), dates.String(), chapter.GetUrl())

			if chapter.GetContent() != "" {
				if foundLessonPlan {
					t.Error("found more than one lesson plan!")
				}

				tel.ReportDebug("^ was today's lesson plan")
				foundLessonPlan = true
			}
		}
		if !foundLessonPlan {
			t.Error("could not find lesson plan for", course.GetName())
		}
	}
}
