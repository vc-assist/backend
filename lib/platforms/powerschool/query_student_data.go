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

func (c *Client) GetStudentData(ctx context.Context, input *GetStudentDataInput) (*StudentData, error) {
	res := &StudentData{}
	err := graphqlQuery(ctx, c.http, "AllStudentData", studentDataQuery, input, res)
	return res, err
}
