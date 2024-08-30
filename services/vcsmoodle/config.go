package vcsmoodle

type Config struct {
	// credentials to the moodle user that is a
	// part of all the classes
	Username string `json:"username"`
	Password string `json:"password"`
}
