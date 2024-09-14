package edit

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
)

type directive string

const (
	action_sets   directive = "sets"
	action_keep             = "keep"
	action_add              = "add"
	action_delete           = "del"
)

type actionLine struct {
	directive directive
	keyLeft   string
	keyRight  string
	comment   string
}

func (l actionLine) String() string {
	result := fmt.Sprintf(`%s "%s" "%s"`, l.directive, l.keyLeft, l.keyRight)
	if l.comment != "" {
		result += " # " + l.comment
	}
	return result
}

const edit_instructions = `# This is a file where you can describe multiple
# edit actions at the same time.
# 
# The format goes:
# <action> "<%[1]s key>" "<%[2]s key>"
#
# Where <action> can be:
# 'keep' = Keep this explicit link (aka. 'do nothing').
# 'add' = Add this explicit link.
# 'del' = Delete this explicit link.
#
# When you want to apply the actions in this file, run:
# 
# linker-cli link edit apply path/to/file.txt

sets "%[1]s" "%[2]s"

`

type actionFile struct {
	leftSet  string
	rightSet string
	actions  []actionLine
}

func (l actionFile) String() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf(edit_instructions, l.leftSet, l.rightSet))
	for _, line := range l.actions {
		builder.WriteString(line.String() + "\n")
	}
	return builder.String()
}

var lineRegex = regexp.MustCompile(`^(\w+)\s*"([^"]+)"\s*"([^"]+)".*$`)

func parseLine(line string) (actionLine, bool, error) {
	line = strings.Trim(line, " \t")
	if len(line) == 0 {
		return actionLine{}, false, nil
	}
	if line[0] == '#' {
		return actionLine{}, false, nil
	}

	matches := lineRegex.FindStringSubmatch(line)
	if len(matches) == 0 {
		return actionLine{}, false, fmt.Errorf("line did not match regex")
	}

	return actionLine{
		directive: directive(matches[1]),
		keyLeft:   matches[2],
		keyRight:  matches[3],
	}, true, nil
}

func newActionFile(reader io.Reader) (actionFile, error) {
	s := bufio.NewScanner(reader)

	leftSet := ""
	rightSet := ""
	var actions []actionLine

	for i := 0; s.Scan(); i++ {
		line, ok, err := parseLine(s.Text())
		if err != nil {
			return actionFile{}, fmt.Errorf(
				"%v (error: line %d)",
				err, i+1,
			)
		}
		if !ok {
			continue
		}

		if line.directive == action_sets {
			leftSet = line.keyLeft
			rightSet = line.keyRight
			continue
		}

		actions = append(actions, line)
	}

	return actionFile{
		leftSet:  leftSet,
		rightSet: rightSet,
		actions:  actions,
	}, nil
}
