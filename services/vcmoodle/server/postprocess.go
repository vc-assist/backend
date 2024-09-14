package server

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
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
		if a.proto.GetDates()[0] < b.proto.GetDates()[0] {
			return -1
		}
		if a.proto.GetDates()[0] > b.proto.GetDates()[0] {
			return 1
		}
		return 0
	})

	now := timezone.Now().Unix()

	var lastValue chapterPair
	for _, c := range chapters {
		allFuture := true
		for _, d := range c.proto.GetDates() {
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

func pbResourceType(resourceType db.ResourceType) vcmoodlev1.ResourceType {
	switch resourceType {
	case db.RESOURCE_GENERIC:
		return vcmoodlev1.ResourceType_GENERIC_URL
	case db.RESOURCE_FILE:
		return vcmoodlev1.ResourceType_FILE
	case db.RESOURCE_BOOK:
		return vcmoodlev1.ResourceType_BOOK
	case db.RESOURCE_HTML_AREA:
		return vcmoodlev1.ResourceType_HTML_AREA
	default:
		return -1
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

				resourceType := pbResourceType(db.ResourceType(resource.Type))
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

					urlStr := ""
					if resource.ID.Valid {
						urlStr = fmt.Sprintf(
							"https://learn.vcs.net/mod/book/view.php?id=%d&chapterid=%d",
							resource.ID.Int64, chapter.ID,
						)
					}
					pbChapter := &vcmoodlev1.Chapter{
						Id:    int64(chapter.ID),
						Name:  chapter.Name,
						Dates: unixTimes,
						Url:   urlStr,
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

		segments := strings.Split(course.Name, " - ")
		name := segments[0]
		teacher := ""
		if len(segments) > 1 {
			teacher = segments[1]
		}

		outCourses[i] = &vcmoodlev1.Course{
			Id:       int64(course.ID),
			Name:     name,
			Teacher:  teacher,
			Url:      fmt.Sprintf("https://learn.vcs.net/course/view.php?id=%d", course.ID),
			Sections: outSections,
		}
	}

	return outCourses, nil
}
