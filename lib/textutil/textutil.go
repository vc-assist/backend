package textutil

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
