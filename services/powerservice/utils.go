package powerservice

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"
	"vcassist-backend/lib/oauth"
	"vcassist-backend/lib/scrapers/powerschool"
	scraper "vcassist-backend/lib/scrapers/powerschool"
	"vcassist-backend/lib/timezone"
	keychainv1 "vcassist-backend/proto/vcassist/services/keychain/v1"
	powerservicev1 "vcassist-backend/proto/vcassist/services/powerservice/v1"
)

type OAuthConfig struct {
	BaseLoginUrl string
	RefreshUrl   string
	ClientId     string
}

func (o OAuthConfig) GetOAuthFlow() (*keychainv1.OAuthFlow, error) {
	codeVerifier, err := oauth.GenerateCodeVerifier()
	if err != nil {
		return nil, err
	}
	return &keychainv1.OAuthFlow{
		BaseLoginUrl:    o.BaseLoginUrl,
		AccessType:      "offline",
		Scope:           "openid email profile",
		RedirectUri:     "com.powerschool.portal://",
		CodeVerifier:    codeVerifier,
		ClientId:        o.ClientId,
		TokenRequestUrl: "https://oauth2.googleapis.com/token",
	}, nil
}

func GetCurrentWeek(now time.Time) (start time.Time, stop time.Time) {
	start = now.Add(-time.Hour * 24 * time.Duration(now.Weekday()))
	stop = now.Add(time.Hour * 24 * time.Duration(time.Saturday-now.Weekday()))
	return start, stop
}

func Scrape(ctx context.Context, client *powerschool.Client) (*powerservicev1.GetStudentDataResponse, error) {
	allStudents, err := client.GetAllStudents(ctx)
	if err != nil {
		return nil, err
	}
	if len(allStudents.Profiles) == 0 {
		return nil, fmt.Errorf(
			"could not find student profile, are your credentials expired?",
		)
	}

	psStudent := allStudents.Profiles[0]
	studentData, err := client.GetStudentData(ctx, scraper.GetStudentDataRequest{
		Guid: psStudent.Guid,
	})
	if err != nil {
		return nil, err
	}

	// MAY BE USED LATER, DO NOT DELETE
	// studentPhoto, err := client.GetStudentPhoto(ctx, scraper.GetStudentPhotoRequest{
	// 	Guid: psStudent.Guid,
	// })
	// if err != nil {
	// 	span.RecordError(err)
	// 	span.SetStatus(codes.Error, "failed to get student photo")
	// }

	gpa, err := strconv.ParseFloat(psStudent.CurrentGpa, 32)
	if err != nil {
		slog.WarnContext(ctx, "failed to parse gpa", "gpa", psStudent.CurrentGpa, "err", err)
	}

	if len(studentData.Student.Courses) == 0 {
		slog.WarnContext(ctx, "student data unavailable, only returning profile...")
		return &powerservicev1.GetStudentDataResponse{
			Profile: &powerservicev1.StudentProfile{
				Guid:       psStudent.Guid,
				CurrentGpa: float32(gpa),
				FirstName:  psStudent.FirstName,
				LastName:   psStudent.LastName,
				// photo is disabled for now as it doesn't have a use
				// Photo: "",
			},
		}, nil
	}

	courses := transformCourses(ctx, studentData.Student.Courses)

	if len(courses) > 0 {
		guids := make([]string, len(courses))
		for i, course := range courses {
			guids[i] = course.Guid
		}

		start, stop := GetCurrentWeek(timezone.Now())
		res, err := client.GetCourseMeetingList(ctx, scraper.GetCourseMeetingListRequest{
			CourseGuids: guids,
			Start:       start.Format(time.RFC3339),
			Stop:        stop.Format(time.RFC3339),
		})
		if err != nil {
			slog.WarnContext(
				ctx,
				"failed to fetch course meetings",
				"err", err,
			)
		}

		transformCourseMeetings(ctx, courses, res.Meetings)
	}

	schools := transformSchools(psStudent.Schools)
	bulletins := transformBulletins(psStudent.Bulletins)

	return &powerservicev1.GetStudentDataResponse{
		Profile: &powerservicev1.StudentProfile{
			Guid:       psStudent.Guid,
			CurrentGpa: float32(gpa),
			FirstName:  psStudent.FirstName,
			LastName:   psStudent.LastName,
		},
		Schools:   schools,
		Bulletins: bulletins,
		Courses:   courses,
	}, nil
}
