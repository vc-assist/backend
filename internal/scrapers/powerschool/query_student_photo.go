package powerschool

const query_student_photo = `query StudentPhoto($guid: ID!) {
  studentPhoto(guid: $guid) {
    image
  }
}`

type requestStudentPhoto struct {
	Guid string `json:"guid"`
}

type responseStudentPhoto struct {
	StudentPhoto struct {
		Image string `json:"image"`
	} `json:"studentPhoto"`
}
