package server

import (
	"testing"
	"time"
	"vcassist-backend/lib/timezone"
	vcmoodlev1 "vcassist-backend/proto/vcassist/services/vcmoodle/v1"
)

func TestLessonPlanResolution(t *testing.T) {
	type chapter struct {
		id   int64
		name string
	}

	cases := []struct {
		now             time.Time
		chapters        []chapter
		expectChapterId int64
	}{
		{
			now: time.Date(timezone.Now().Year(), time.September, 8, 0, 0, 0, 0, timezone.Location),
			chapters: []chapter{
				{id: 0, name: "August 14"},
				{id: 1, name: "August 16"},
				{id: 2, name: "August 20"},
				{id: 3, name: "August 26"},
				{id: 4, name: "August 22"},
				{id: 5, name: "August 28"},
				{id: 6, name: "Sept 3"},
				{id: 7, name: "September 5"},
			},
			expectChapterId: 7,
		},
	}

	for _, test := range cases {
		chapterPairs := make([]chapterPair, len(test.chapters))
		for i, chapter := range test.chapters {
			dates, err := parseTOCDate(chapter.name)
			if err != nil {
				t.Fatal(err)
			}
			unixDates := make([]int64, len(dates))
			for i, d := range dates {
				unixDates[i] = d.Unix()
			}

			chapterPairs[i] = chapterPair{
				proto: &vcmoodlev1.Chapter{
					Id:    chapter.id,
					Name:  chapter.name,
					Dates: unixDates,
				},
				contentHtml: "CONTENT",
			}
		}

		setLessonPlanChapter(chapterPairs)

		var resolved chapterPair
		for _, chapter := range chapterPairs {
			if chapter.proto.HomepageContent != "" {
				resolved = chapter
				break
			}
		}

		if (resolved == chapterPair{}) {
			t.Fatal(
				"failed to resolve any chapter to be the lesson plan",
				"now", test.now.Format(time.RFC850), "expected", test.expectChapterId, "to be resolved",
			)
		}
		if resolved.proto.Id != test.expectChapterId {
			t.Fatal(
				"resolved the wrong chapter",
				"now", test.now.Format(time.RFC850), "expected", test.expectChapterId,
				"instead got", resolved.proto.Id,
			)
		}
	}
}
