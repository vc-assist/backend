package apis

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	moodlev1 "vcassist-backend/api/vcassist/moodle/v1"
	"vcassist-backend/internal/db"
	"vcassist-backend/lib/timezone"

	"google.golang.org/protobuf/types/known/timestamppb"
)

var referenceMonths = []string{
	"january",
	"february",
	"march",
	"april",
	"may",
	"june",
	"july",
	"august",
	"september",
	"october",
	"november",
	"december",
}

func parseMonth(text string) time.Month {
	text = strings.ToLower(text)
	for i, month := range referenceMonths {
		if strings.Contains(month, text) {
			return time.January + time.Month(i)
		}
	}
	return -1
}

func resolveTOCMonthDay(month time.Month, day int) time.Time {
	now := timezone.Now()
	year := now.Year()

	if (month >= time.August && month <= time.December) &&
		(now.Month() < time.June && now.Month() >= time.January) {
		year--
	}
	if (month >= time.January && month < time.June) &&
		(now.Month() >= time.August && now.Month() <= time.December) {
		year++
	}

	return time.Date(year, month, day, 0, 0, 0, 0, timezone.Location)
}

var monthDayRegex = regexp.MustCompile(`([A-Za-z]{3,9}) *(\d{1,2})`)
var monthDayDayRegex = regexp.MustCompile(`(\w+) *(\d{1,2}) *[^\d\w\s] *(\d{1,2})(?:[^\d]|$)`)

func parseTOCDate(text string) ([]time.Time, error) {
	monthDayDayMatch := monthDayDayRegex.FindStringSubmatch(text)
	if len(monthDayDayMatch) >= 4 {
		month := parseMonth(monthDayDayMatch[1])
		day1, err := strconv.ParseInt(monthDayDayMatch[2], 10, 32)
		if err != nil {
			return nil, err
		}
		day2, err := strconv.ParseInt(monthDayDayMatch[3], 10, 32)
		if err != nil {
			return nil, err
		}

		return []time.Time{
			resolveTOCMonthDay(month, int(day1)),
			resolveTOCMonthDay(month, int(day2)),
		}, nil
	}

	monthDayMatches := monthDayRegex.FindAllStringSubmatch(text, -1)
	var dates []time.Time
	for _, match := range monthDayMatches {
		if len(match) < 3 {
			continue
		}
		month := parseMonth(match[1])
		day, err := strconv.ParseInt(match[2], 10, 32)
		if err != nil {
			slog.Warn("failed to parse day", "matches", match, "err", err)
			continue
		}
		dates = append(dates, resolveTOCMonthDay(month, int(day)))
	}
	return dates, nil
}

// QueryLessonPlans implements the interface method.
func (m MoodleImpl) QueryLessonPlans(ctx context.Context, courseIds []int64) (*moodlev1.LessonPlansResponse, error) {
	dbCourses, err := m.db.GetMoodleCourses(ctx, courseIds)
	if err != nil {
		m.tel.ReportBroken(report_db_query, err, "GetCourses", courseIds)
		return nil, err
	}

	courses := make([]*moodlev1.LessonPlansResponse_Course, len(dbCourses))
	for i, course := range dbCourses {
		segments := strings.Split(course.Name, " - ")
		name := segments[0]
		teacher := ""
		if len(segments) > 1 {
			teacher = segments[1]
		}

		courses[i] = &moodlev1.LessonPlansResponse_Course{
			Id:      course.ID,
			Name:    name,
			Teacher: teacher,
			Url:     fmt.Sprintf("https://learn.vcs.net/course/view.php?id=%d", course.ID),
		}

		var chapters []*moodlev1.LessonPlansResponse_Chapter

		dbSections, err := m.db.GetMoodleCourseSections(ctx, course.ID)
		if err != nil {
			m.tel.ReportBroken(report_db_query, err, "GetCourseSections", course.ID)
			continue
		}
		for _, section := range dbSections {
			dbResources, err := m.db.GetMoodleSectionResources(ctx, db.GetMoodleSectionResourcesParams{
				CourseID:   course.ID,
				SectionIdx: section.Idx,
			})
			if err != nil {
				m.tel.ReportBroken(report_db_query, err, "GetSectionResources", course.ID, section.Idx)
				continue
			}
			for _, resource := range dbResources {
				dbChapters, err := m.db.GetMoodleResourceChapters(ctx, db.GetMoodleResourceChaptersParams{
					CourseID:    course.ID,
					SectionIdx:  section.Idx,
					ResourceIdx: resource.Idx,
				})
				if err != nil {
					m.tel.ReportBroken(report_db_query, err, "GetResourceChapters", course.ID, section.Idx, resource.Idx)
					continue
				}

				for _, chapter := range dbChapters {
					times, err := parseTOCDate(chapter.Name)
					if err != nil {
						m.tel.ReportBroken(report_moodle_parse_toc_date, err, chapter.Name)
						continue
					}

					dates := make([]*timestamppb.Timestamp, len(times))
					for i, t := range times {
						dates[i] = timestamppb.New(t)
					}

					urlStr := ""
					if resource.ID.Valid {
						urlStr = fmt.Sprintf(
							"https://learn.vcs.net/mod/book/view.php?id=%d&chapterid=%d",
							resource.ID.Int64, chapter.ID,
						)
					} else {
						m.tel.ReportBroken(
							report_moodle_query_lesson_plans,
							fmt.Errorf("resource id null"),
							resource.Url,
							resource.CourseID,
							resource.SectionIdx,
						)
					}

					chapters = append(chapters, &moodlev1.LessonPlansResponse_Chapter{
						Id:    chapter.ID,
						Dates: dates,
						Url:   urlStr,
					})
				}
			}
		}

		slices.SortFunc(chapters, func(a, b *moodlev1.LessonPlansResponse_Chapter) int {
			if a.Dates[0].Seconds < b.Dates[0].Seconds {
				return -1
			}
			if a.Dates[0].Seconds > b.Dates[0].Seconds {
				return 1
			}
			return 0
		})

		now := timezone.Now().Unix()

		var currentChapter *moodlev1.LessonPlansResponse_Chapter
		for _, c := range chapters {
			allFuture := true
			for _, d := range c.Dates {
				if now > d.Seconds {
					allFuture = false
					break
				}
			}
			if allFuture {
				break
			}
			currentChapter = c
		}
		if currentChapter == nil {
			m.tel.ReportWarning(
				report_moodle_query_lesson_plans,
				fmt.Errorf("lesson plan not found"),
				courses[i].Url,
			)
			continue
		}

		content, err := m.db.GetMoodleChapterContent(ctx, currentChapter.Id)
		if err != nil {
			m.tel.ReportBroken(report_db_query, err, "GetChapterContent", currentChapter.Id)
			continue
		}

		currentChapter.Content = &content
	}

	return &moodlev1.LessonPlansResponse{
		Courses: courses,
	}, nil
}

// QueryChapterContent implements the interface method.
func (m MoodleImpl) QueryChapterContent(ctx context.Context, chapterId int64) (string, error) {
	content, err := m.db.GetMoodleChapterContent(ctx, chapterId)
	if err != nil {
		m.tel.ReportBroken(report_db_query, err, "GetChapterContent", chapterId)
		return "", err
	}
	return content, nil
}

// QueryUserCourseIds implements the interface method.
func (m MoodleImpl) QueryUserCourseIds(ctx context.Context, accountId int64) ([]int64, error) {
	courseIds, err := m.db.GetMoodleUserCourses(ctx, accountId)
	if err != nil {
		m.tel.ReportBroken(report_db_query, err, "GetUserCourses", accountId)
		return nil, err
	}
	return courseIds, nil
}
