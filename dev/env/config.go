package devenv

type ViewMoodleTestConfig struct {
	TargetCourse string `json:"target_course"`
}

type EditMoodleTestConfig struct {
	TargetCourse string `json:"target_course"`
}

type MoodleTestConfig struct {
	BaseUrl    string               `json:"base_url"`
	Username   string               `json:"username"`
	Password   string               `json:"password"`
	ViewConfig ViewMoodleTestConfig `json:"view"`
	EditConfig EditMoodleTestConfig `json:"edit"`
}
