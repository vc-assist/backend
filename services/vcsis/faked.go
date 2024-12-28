package vcsis

import (
	"math/rand"
	"time"
	sisv1 "vcassist-backend/proto/vcassist/services/sis/v1"

	"github.com/bxcodec/faker/v4"
)

func generateMockData() *sisv1.Data {
	return &sisv1.Data{
		Profile:   generateMockStudentProfile(),
		Schools:   generateMockSchoolData(3),
		Bulletins: generateMockBulletins(5),
		Courses:   generateMockCourseData(4),
	}
}

func generateMockStudentProfile() *sisv1.StudentProfile {
	return &sisv1.StudentProfile{
		Guid:       faker.UUIDHyphenated(),
		CurrentGpa: randomFloat(0, 4),
		Name:       faker.Name(),
		Photo:      []byte(faker.Word()), // Mock photo bytes
	}
}

func generateMockSchoolData(count int) []*sisv1.SchoolData {
	var schools []*sisv1.SchoolData
	for i := 0; i < count; i++ {
		schools = append(schools, &sisv1.SchoolData{
			Name:          faker.Name(),
			Phone:         faker.Phonenumber(),
			Fax:           faker.Phonenumber(),
			Email:         faker.Email(),
			StreetAddress: "",
			City:          "",
			State:         "",
			Zip:           "",
			Country:       "",
		})
	}
	return schools
}

func generateMockBulletins(count int) []*sisv1.Bulletin {
	var bulletins []*sisv1.Bulletin
	for i := 0; i < count; i++ {
		bulletins = append(bulletins, &sisv1.Bulletin{
			Title:     faker.Word(),
			StartDate: time.Now().Unix(),
			EndDate:   time.Now().AddDate(0, 0, rand.Intn(10)+1).Unix(),
			Body:      faker.Paragraph(),
		})
	}
	return bulletins
}

func generateMockCourseData(count int) []*sisv1.CourseData {
	var courses []*sisv1.CourseData
	for i := 0; i < count; i++ {
		courses = append(courses, &sisv1.CourseData{
			Guid:                 faker.UUIDHyphenated(),
			Name:                 faker.Word(),
			Period:               faker.Word(),
			Teacher:              faker.Name(),
			TeacherEmail:         faker.Email(),
			Room:                 faker.Word(),
			OverallGrade:         randomFloat(0, 100),
			DayName:              "A",
			HomeworkPasses:       rand.Int31n(10),
			Assignments:          generateMockAssignments(5),
			Meetings:             generateMockMeetings(3),
			Snapshots:            generateMockGradeSnapshots(10),
			AssignmentCategories: generateMockAssignmentCategories(3),
		})
	}
	return courses
}

func generateMockAssignments(count int) []*sisv1.AssignmentData {
	var assignments []*sisv1.AssignmentData
	pointsEarned := randomFloat(0, 100)
	pointsPossible := randomFloat(100, 200)
	for i := 0; i < count; i++ {
		assignments = append(assignments, &sisv1.AssignmentData{
			Title:          faker.Word(),
			Description:    faker.Paragraph(),
			Category:       faker.Word(),
			DueDate:        time.Now().AddDate(0, 0, rand.Intn(10)+1).Unix(),
			PointsEarned:   &pointsEarned,
			PointsPossible: &pointsPossible,
			IsMissing:      randomBool(),
			IsLate:         randomBool(),
			IsCollected:    randomBool(),
			IsExempt:       randomBool(),
			IsIncomplete:   randomBool(),
		})
	}
	return assignments
}

func generateMockMeetings(count int) []*sisv1.Meeting {
	var meetings []*sisv1.Meeting
	for i := 0; i < count; i++ {
		start := time.Now().Add(time.Duration(rand.Intn(24)) * time.Hour).Unix()
		stop := start + int64(rand.Intn(3600)+1800)
		meetings = append(meetings, &sisv1.Meeting{
			Start: start,
			Stop:  stop,
		})
	}
	return meetings
}

func generateMockGradeSnapshots(count int) []*sisv1.GradeSnapshot {
	var snapshots []*sisv1.GradeSnapshot
	for i := 0; i < count; i++ {
		snapshots = append(snapshots, &sisv1.GradeSnapshot{
			Time:  time.Now().AddDate(0, 0, -rand.Intn(30)).Unix(),
			Value: randomFloat(0, 100),
		})
	}
	return snapshots
}

func generateMockAssignmentCategories(count int) []*sisv1.AssignmentCategory {
	var categories []*sisv1.AssignmentCategory
	for i := 0; i < count; i++ {
		categories = append(categories, &sisv1.AssignmentCategory{
			Name:   faker.Word(),
			Weight: randomFloat(0, 1),
		})
	}
	return categories
}

// Helper functions
func randomFloat(min, max float64) float32 {
	return float32(min + rand.Float64()*(max-min))
}

func randomBool() bool {
	return rand.Intn(2) == 0
}
