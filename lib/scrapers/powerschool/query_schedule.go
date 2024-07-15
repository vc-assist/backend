package powerschool

import (
	"context"
	powerschoolv1 "vcassist-backend/proto/vcassist/scrapers/powerschool/v1"
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

func (c *Client) GetCourseMeetingList(ctx context.Context, input *powerschoolv1.GetCourseMeetingListInput) (*powerschoolv1.CourseMeetingList, error) {
	res := &powerschoolv1.CourseMeetingList{}
	err := graphqlQuery(ctx, c.http, "SectionMeetings", scheduleQuery, input, res)
	return res, err
}
