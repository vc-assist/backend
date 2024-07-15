package powerschool

import (
	"context"
	powerschoolv1 "vcassist-backend/proto/vcassist/scrapers/powerschool/v1"
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

func (c *Client) GetStudentData(ctx context.Context, input *powerschoolv1.GetStudentDataInput) (*powerschoolv1.StudentData, error) {
	res := &powerschoolv1.StudentData{}
	err := graphqlQuery(ctx, c.http, "AllStudentData", studentDataQuery, input, res)
	return res, err
}
