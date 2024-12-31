package powerschool

import (
	"context"
)

const studentPhotoQuery = `query StudentPhoto($guid: ID!) {
  studentPhoto(guid: $guid) {
    image
  }
}`

type getStudentPhotoRequest struct {
	Guid string `json:"guid"`
}

type getStudentPhotoResponse struct {
	StudentPhoto struct {
		Image string `json:"image"`
	} `json:"studentPhoto"`
}

func (c *client) GetStudentPhoto(ctx context.Context, req getStudentPhotoRequest) (*getStudentPhotoResponse, error) {
	res := &getStudentPhotoResponse{}
	err := graphqlQuery(
		ctx, c, "StudentPhoto", studentPhotoQuery,
		req, res,
	)
	return res, err
}
