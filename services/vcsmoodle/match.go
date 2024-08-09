package vcsmoodle

import (
	"regexp"
	"strings"
)

var whitespaceRegex = regexp.MustCompile(`\s+`)

func NormalizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.Trim(name, " \n\t")
	name = whitespaceRegex.ReplaceAllString(name, "")
	return name
}

func MatchName(name string, matchers []string) bool {
	name = NormalizeName(name)
	for _, m := range matchers {
		if strings.Contains(name, m) {
			return true
		}
	}
	return false
}

var blacklistCourseKeywords = []string{
	"christianservice",
	"ipad",
	"homepage",
	"impact",
	"college",
	"vcs",
	"vcs",
	"vcjh",
	"vcassist",
	"athletic",
	"deca",
	"sat/actprep",
	"pianoii",
	"pianoiii",
	"studyhall",
	"amse",
	"chorale",
	"productionarts",
	"peextension",
	"counselor",
}
var classInfoKeywords = []string{
	"welcome",
	"classinfo",
	"syllabus",
}
var lessonPlanKeywords = []string{
	"lessonplan",
	"homework",
	"classwork",
}
var zoomKeywords = []string{
	"meetingid",
	"zoomroom",
	"zoomcode",
}
