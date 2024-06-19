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

func (c *Client) GetCourseMeetingList(ctx context.Context, input *GetCourseMeetingListInput) (*CourseMeetingList, error) {
	return graphqlQuery[*GetCourseMeetingListInput, *CourseMeetingList](
		ctx, c.http, "SectionMeetings", scheduleQuery, input,
	)
}
