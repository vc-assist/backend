package powerschool

import (
	"context"
)

const studentDataQuery = `query AllStudentData($guid: ID!) {
  student(guid: $guid) {
    sections {
      ...sectionData
    }
  }
}
fragment sectionData on SectionType {
  guid
  name
  period
  teacherFirstName
  teacherLastName
  teacherEmail
  assignments {
    ...assignmentData
  }
  terms {
    ...termData
  }
  room
}
fragment assignmentData on AssignmentType {
  title
  category
  description
  dueDate
  pointsEarned
  pointsPossible
  attributeMissing
  attributeLate
  attributeCollected
  attributeExempt
  attributeIncomplete
}
fragment termData on TermType {
  start
  end
  finalGrade {
    ...finalGradeData
  }
}
fragment finalGradeData on FinalGradeType {
  percent
  inProgressStatus
}`

type finalGrade struct {
	Percent          int  `json:"percent"`
	InProgressStatus bool `json:"inProgressStatus"`
}

type termData struct {
	Start      string     `json:"start"`
	End        string     `json:"end"`
	FinalGrade finalGrade `json:"finalGrade"`
}

type assignmentData struct {
	Title               string   `json:"title"`
	Category            string   `json:"category"`
	DueDate             string   `json:"dueDate"`
	Description         string   `json:"description"`
	PointsEarned        *float32 `json:"pointsEarned"`
	PointsPossible      *float32 `json:"pointsPossible"`
	AttributeMissing    bool     `json:"attributeMissing"`
	AttributeLate       bool     `json:"attributeLate"`
	AttributeCollected  bool     `json:"attributeCollected"`
	AttributeExempt     bool     `json:"attributeExempt"`
	AttributeIncomplete bool     `json:"attributeIncomplete"`
}

type courseData struct {
	Guid             string           `json:"guid"`
	Name             string           `json:"name"`
	Period           string           `json:"period"`
	TeacherFirstName string           `json:"teacherFirstName"`
	TeacherLastName  string           `json:"teacherLastName"`
	TeacherEmail     string           `json:"teacherEmail"`
	Room             string           `json:"room"`
	Assignments      []assignmentData `json:"assignments"`
	Terms            []termData       `json:"terms"`
}

type getStudentDataResponse struct {
	Student struct {
		Courses []courseData `json:"sections"`
	} `json:"student"`
}

type getStudentDataRequest struct {
	Guid string `json:"guid"`
}

func (c *client) GetStudentData(ctx context.Context, req getStudentDataRequest) (*getStudentDataResponse, error) {
	res := &getStudentDataResponse{}
	err := graphqlQuery(
		ctx, c, "AllStudentData", studentDataQuery,
		req, res,
	)
	return res, err
}
