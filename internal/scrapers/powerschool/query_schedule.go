package powerschool

import (
	"context"
)

const query_schedule = `query SectionMeetings(
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

type courseMeeting struct {
	CourseGuid string `json:"sectionGuid"`
	Start      string `json:"start"`
	Stop       string `json:"stop"`
}

type getCourseMeetingListResponse struct {
	Meetings []courseMeeting `json:"sectionMeetings"`
}

type getCourseMeetingListRequest struct {
	CourseGuids []string `json:"sectionGuids"`
	// ISO timestamp
	Start string `json:"start"`
	// ISO timestamp
	Stop string `json:"stop"`
}

func (c *client) GetCourseMeetingList(ctx context.Context, req getCourseMeetingListRequest) (*getCourseMeetingListResponse, error) {
	res := &getCourseMeetingListResponse{}
	err := graphqlQuery(
		ctx, c, "SectionMeetings", query_schedule,
		req, res,
	)
	return res, err
}
