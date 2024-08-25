package powerschool

import (
	"context"
)

const studentPhotoQuery = `query StudentPhoto($guid: ID!) {
  studentPhoto(guid: $guid) {
    image
  }
}`

type GetStudentPhotoRequest struct {
	Guid string `json:"guid"`
}

type GetStudentPhotoResponse struct {
	StudentPhoto struct {
		Image string `json:"image"`
	} `json:"studentPhoto"`
}

func (c *Client) GetStudentPhoto(ctx context.Context, req GetStudentPhotoRequest) (*GetStudentPhotoResponse, error) {
	res := &GetStudentPhotoResponse{}
	err := graphqlQuery(
		ctx, c.http, "StudentPhoto", studentPhotoQuery,
		req, res,
	)
	return res, err
}
