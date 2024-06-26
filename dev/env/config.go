package devenv

type MoodleTestConfig struct {
	BaseUrl        string `json:"base_url"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	SpecificCourse string `json:"specific_course"`
}

