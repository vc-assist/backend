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
	SectionGuid string `json:"sectionGuid"`
	Start       string `json:"start"`
	Stop        string `json:"stop"`
}

type CourseMeetingList = []CourseMeeting

type GetCourseMeetingListInput struct {
	CourseIds []string `json:"sectionGuids"`
	Start     string   `json:"start"`
	Stop      string   `json:"stop"`
}

func (c *Client) GetCourseMeetingList(ctx context.Context, input GetCourseMeetingListInput) (CourseMeetingList, error) {
	return graphqlQuery[GetCourseMeetingListInput, CourseMeetingList](
		ctx, c.http, "SectionMeetings", scheduleQuery, input,
	)
}
