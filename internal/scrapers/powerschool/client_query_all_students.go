package powerschool

import (
	"context"
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

type schoolData struct {
	Name          string `json:"name"`
	Phone         string `json:"phone"`
	Fax           string `json:"fax"`
	Email         string `json:"email"`
	StreetAddress string `json:"streetAddress"`
	City          string `json:"city"`
	State         string `json:"state"`
	Zip           string `json:"zip"`
	Country       string `json:"country"`
}

type bulletin struct {
	Title     string `json:"title"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Body      string `json:"body"`
}

type studentProfile struct {
	Guid       string       `json:"guid"`
	CurrentGpa string       `json:"currentGPA"`
	FirstName  string       `json:"firstName"`
	LastName   string       `json:"lastName"`
	Schools    []schoolData `json:"schools"`
	Bulletins  []bulletin   `json:"bulletins"`
}

type getAllStudentsResponse struct {
	Profiles []studentProfile `json:"students"`
}

func (c *client) GetAllStudents(ctx context.Context) (*getAllStudentsResponse, error) {
	res := &getAllStudentsResponse{}
	err := graphqlQuery(
		ctx, c, "AllStudentsFirstLevel", allStudentsQuery,
		struct{}{}, res,
	)
	return res, err
}
