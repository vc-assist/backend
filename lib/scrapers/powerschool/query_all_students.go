package powerschool

import (
	"context"
	powerschoolv1 "vcassist-backend/proto/vcassist/scrapers/powerschool/v1"
)

const allStudentsQuery = `query AllStudentsFirstLevel {
  students {
    ...studentData
  }
}
fragment studentData on StudentType {
  guid
  firstName
  lastName
  schools {
    ...schoolData
  }
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

func (c *Client) GetAllStudents(ctx context.Context) (*powerschoolv1.AllStudents, error) {
	res := &powerschoolv1.AllStudents{}
	err := graphqlQuery(ctx, c.http, "AllStudentsFirstLevel", allStudentsQuery, &powerschoolv1.GetAllStudentsInput{}, res)
	return res, err
}
