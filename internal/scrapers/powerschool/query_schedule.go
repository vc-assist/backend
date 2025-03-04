package powerschool

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

type responseSchedule struct {
	Meetings []courseMeeting `json:"sectionMeetings"`
}

type requestSchedule struct {
	CourseGuids []string `json:"sectionGuids"`
	// ISO timestamp
	Start string `json:"start"`
	// ISO timestamp
	Stop string `json:"stop"`
}
