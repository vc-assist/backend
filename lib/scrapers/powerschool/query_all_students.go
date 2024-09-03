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

type SchoolData struct {
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

type Bulletin struct {
	Title     string `json:"title"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Body      string `json:"body"`
}

type StudentProfile struct {
	Guid       string       `json:"guid"`
	CurrentGpa string       `json:"currentGPA"`
	FirstName  string       `json:"firstName"`
	LastName   string       `json:"lastName"`
	Schools    []SchoolData `json:"schools"`
	Bulletins  []Bulletin   `json:"bulletins"`
}

type GetAllStudentsResponse struct {
	Profiles []StudentProfile `json:"students"`
}

func (c *Client) GetAllStudents(ctx context.Context) (*GetAllStudentsResponse, error) {
	res := &GetAllStudentsResponse{}
	err := graphqlQuery(
		ctx, c.http, "AllStudentsFirstLevel", allStudentsQuery,
		struct{}{}, res,
	)
	return res, err
}
