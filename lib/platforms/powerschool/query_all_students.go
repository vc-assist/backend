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

type Bulletin struct {
	Title     string `json:"title"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Body      string `json:"body"`
}

type StudentProfile struct {
	Guid       string     `json:"guid"`
	CurrentGpa string     `json:"currentGPA"`
	FirstName  string     `json:"firstName"`
	LastName   string     `json:"lastName"`
	GradeLevel int        `json:"gradeLevel"`
	Bulletins  []Bulletin `json:"bulletins"`
}

type AllStudents struct {
	Students []StudentProfile `json:"students"`
}

func (c *Client) GetAllStudents(ctx context.Context) (AllStudents, error) {
	return graphqlQuery[struct{}, AllStudents](
		ctx, c.http, "AllStudentsFirstLevel", allStudentsQuery, struct{}{},
	)
}
