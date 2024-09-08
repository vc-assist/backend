package server

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"vcassist-backend/lib/timezone"
	vcmoodlev1 "vcassist-backend/proto/vcassist/services/vcmoodle/v1"
	"vcassist-backend/services/vcmoodle/db"
)

type chapterPair struct {
	proto       *vcmoodlev1.Chapter
	contentHtml string
}

func setLessonPlanChapter(chapters []chapterPair) {
	slices.SortFunc(chapters, func(a, b chapterPair) int {
		if a.proto.Dates[0] < b.proto.Dates[0] {
			return -1
		}
		if a.proto.Dates[0] > b.proto.Dates[0] {
			return 1
		}
		return 0
	})

	now := timezone.Now().Unix()

	var lastValue chapterPair
	for _, c := range chapters {
		allFuture := true
		for _, d := range c.proto.Dates {
			if now > d {
				allFuture = false
				break
			}
		}
		if allFuture {
			break
		}
		lastValue = c
	}
	if lastValue.proto != nil {
		lastValue.proto.HomepageContent = lastValue.contentHtml
	}
}

func GetCourseData(ctx context.Context, qry *db.Queries, dbCourses []db.Course) ([]*vcmoodlev1.Course, error) {
	outCourses := make([]*vcmoodlev1.Course, len(dbCourses))
	for i, course := range dbCourses {
		dbSections, err := qry.GetCourseSections(ctx, course.ID)
		if err != nil {
			return nil, err
		}

		var datedChapters []chapterPair

		outSections := make([]*vcmoodlev1.Section, len(dbSections))
		for i, section := range dbSections {
			dbResources, err := qry.GetSectionResources(ctx, db.GetSectionResourcesParams{
				CourseID:   course.ID,
				SectionIdx: section.Idx,
			})
			if err != nil {
				return nil, err
			}

			outResources := make([]*vcmoodlev1.Resource, len(dbResources))
			for i, resource := range dbResources {
				dbChapters, err := qry.GetResourceChapters(ctx, db.GetResourceChaptersParams{
					CourseID:    course.ID,
					SectionIdx:  section.Idx,
					ResourceIdx: resource.Idx,
				})
				if err != nil {
					return nil, err
				}

				resourceType := pbResourceType(resource.Type)
				if resourceType < 0 {
					slog.WarnContext(ctx, "unknown resource type", "type", resource.Type)
					continue
				}

				outChapters := make([]*vcmoodlev1.Chapter, len(dbChapters))
				for i, chapter := range dbChapters {
					times, err := parseTOCDate(chapter.Name)
					if err != nil {
						slog.WarnContext(
							ctx,
							"parse dates from chapter name",
							"name", chapter.Name,
							"err", err,
						)
					}
					unixTimes := make([]int64, len(times))
					for i, t := range times {
						unixTimes[i] = t.Unix()
					}

					pbChapter := &vcmoodlev1.Chapter{
						Id:    int64(chapter.ID),
						Name:  chapter.Name,
						Dates: unixTimes,
					}
					outChapters[i] = pbChapter

					if len(times) > 0 {
						datedChapters = append(datedChapters, chapterPair{
							proto:       pbChapter,
							contentHtml: chapter.ContentHtml,
						})
					}
				}

				outResources[i] = &vcmoodlev1.Resource{
					Idx:            int64(resource.Idx),
					Type:           resourceType,
					Url:            resource.Url,
					DisplayContent: resource.DisplayContent,
					Chapters:       outChapters,
				}
			}

			outSections[i] = &vcmoodlev1.Section{
				Name:      section.Name,
				Idx:       int64(section.Idx),
				Url:       fmt.Sprintf("https://learn.vcs.net/course/view.php?id=%d&section=%d", course.ID, section.Idx),
				Resources: outResources,
			}
		}

		setLessonPlanChapter(datedChapters)

		outCourses[i] = &vcmoodlev1.Course{
			Id:       int64(course.ID),
			Name:     course.Name,
			Url:      fmt.Sprintf("https://learn.vcs.net/course/view.php?id=%d", course.ID),
			Sections: outSections,
		}
	}

	return outCourses, nil
}
