package powerschool

import "context"

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

type TermData struct {
	Start      string `json:"start"`
	End        string `json:"end"`
	FinalGrade struct {
		Percent          int  `json:"percent"`
		InProgressStatus bool `json:"inProgressStatus"`
	} `json:"finalGrade"`
}

type AssignmentData struct {
	Title               string `json:"title"`
	Category            string `json:"category"`
	DueDate             string `json:"dueDate"`
	Description         string `json:"description"`
	PointsEarned        int    `json:"pointsEarned"`
	PointsPossible      int    `json:"pointsPossible"`
	AttributeMissing    bool   `json:"attributeMissing"`
	AttributeLate       bool   `json:"attributeLate"`
	AttributeCollected  bool   `json:"attributeCollected"`
	AttributeExempt     bool   `json:"attributeExempt"`
	AttributeIncomplete bool   `json:"attributeIncomplete"`
}

type CourseData struct {
	Guid             string           `json:"guid"`
	Name             string           `json:"name"`
	Period           string           `json:"period"`
	TeacherFirstName string           `json:"teacherFirstName"`
	TeacherLastName  string           `json:"teacherLastName"`
	TeacherEmail     string           `json:"teacherEmail"`
	Room             string           `json:"room"`
	Assignments      []AssignmentData `json:"assignments"`
	Terms            []TermData       `json:"terms"`
}

type StudentData struct {
	Student struct {
		Courses []CourseData `json:"sections"`
	} `json:"student"`
}

type GetStudentDataInput struct {
	StudentId string `json:"guid"`
}

func (c *Client) GetStudentData(ctx context.Context, input GetStudentDataInput) (StudentData, error) {
	return graphqlQuery[GetStudentDataInput, StudentData](
		ctx, c.http, "AllStudentData", studentDataQuery, input,
	)
}
