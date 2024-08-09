package powerschool

import (
	"context"
	powerschoolv1 "vcassist-backend/proto/vcassist/scrapers/powerschool/v1"
)

const studentPhotoQuery = `query StudentPhoto($guid: ID!) {
  studentPhoto(guid: $guid) {
    image
  }
}`

func (c *Client) GetStudentPhoto(ctx context.Context, input *powerschoolv1.GetStudentDataInput) (*powerschoolv1.StudentPhoto, error) {
	res := &powerschoolv1.StudentPhoto{}
	err := graphqlQuery(ctx, c.http, "StudentPhoto", studentPhotoQuery, input, res)
	return res, err
}
