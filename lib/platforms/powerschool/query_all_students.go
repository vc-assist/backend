package powerschool

import "context"

const allStudentsQuery = `query AllStudentsFirstLevel {
  students {
    ...studentData
  }
}
fragment studentData on StudentType {
  guid
  firstName
  lastName
  gradeLevel
  bulletins {
    ...bulletinData
  }
  currentGPA
}
fragment schoolData on SchoolType {
  name
  phone
  fax
  email
  streetAddress
  city
  state
  zip
  country
}
fragment bulletinData on BulletinType {
  title
  startDate
  endDate
  body
}`

func (c *Client) GetAllStudents(ctx context.Context) (*AllStudents, error) {
	return graphqlQuery[*GetAllStudentsInput, *AllStudents](
		ctx, c.http, "AllStudentsFirstLevel", allStudentsQuery, &GetAllStudentsInput{},
	)
}
