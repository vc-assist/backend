package vchs

import (
	_ "embed"

	"github.com/titanous/json5"
)

// Map<string, Map<string, float32>>
type weightData map[string]map[string]float32

//go:embed weights.local.json5
var weightsFile []byte

var weights weightData
var weightCourseNames []string

func init() {
	err := json5.Unmarshal(weightsFile, &weights)
	if err != nil {
		panic(err)
	}
	weightCourseNames = make([]string, len(weights))
	i := 0
	for courseName := range weights {
		weightCourseNames[i] = courseName
		i++
	}
}
