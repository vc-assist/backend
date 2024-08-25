package powerschool

import (
	"context"
)

const scheduleQuery = `query SectionMeetings(
  $sectionGuids: [ID]!
  $start: DateTime!
  $stop: DateTime!
) {
  sectionMeetings(sectionGuids: $sectionGuids, start: $start, stop: $stop) {
    ...sectionMeetingData
  }
}
fragment sectionMeetingData on SectionMeetingType {
  sectionGuid
  start
  stop
}`

type CourseMeeting struct {
	CourseGuid string `json:"sectionGuid"`
	Start      string `json:"start"`
	Stop       string `json:"stop"`
}

type GetCourseMeetingListResponse struct {
	Meetings []CourseMeeting `json:"sectionMeetings"`
}

type GetCourseMeetingListRequest struct {
	CourseGuids []string `json:"sectionGuids"`
	// ISO timestamp
	Start string `json:"start"`
	// ISO timestamp
	Stop string `json:"stop"`
}

func (c *Client) GetCourseMeetingList(ctx context.Context, req GetCourseMeetingListRequest) (*GetCourseMeetingListResponse, error) {
	res := &GetCourseMeetingListResponse{}
	err := graphqlQuery(
		ctx, c.http, "SectionMeetings", scheduleQuery,
		req, res,
	)
	return res, err
}
