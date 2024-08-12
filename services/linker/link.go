package linker

import (
	"github.com/antzucaro/matchr"
)

type ImplicitLink struct {
	Left        string
	Right       string
	Correlation float64
}

func CreateImplicitLinks(leftList, rightList []string) []ImplicitLink {
	swapped := false
	if len(rightList) < len(leftList) {
		originalLeftList := leftList
		leftList = rightList
		rightList = originalLeftList
		swapped = true
	}

	var result []ImplicitLink
	matchedLeft := make(map[string]struct{})
	matchedRight := make(map[string]struct{})

	for _, left := range leftList {
		for _, right := range rightList {
			_, isMatchedRight := matchedRight[right]
			if isMatchedRight {
				continue
			}
			if left == right {
				link := ImplicitLink{
					Left:        left,
					Right:       right,
					Correlation: 1,
				}
				if swapped {
					link.Right = left
					link.Left = right
				}

				result = append(result, link)
				matchedLeft[left] = struct{}{}
				matchedRight[right] = struct{}{}
				break
			}
		}
	}

	for _, left := range leftList {
		_, isMatchedLeft := matchedLeft[left]
		if isMatchedLeft {
			continue
		}

		var mostSimilarity float64
		var mostSimilarRight string

		for _, right := range rightList {
			_, isMatchedRight := matchedRight[right]
			if isMatchedRight {
				continue
			}

			similarity := matchr.JaroWinkler(left, right, false)
			if similarity > mostSimilarity {
				mostSimilarity = similarity
				mostSimilarRight = right
			}
		}

		if mostSimilarity > 0 {
			link := ImplicitLink{
				Left:        left,
				Right:       mostSimilarRight,
				Correlation: mostSimilarity,
			}
			if swapped {
				link.Right = left
				link.Left = mostSimilarRight
			}

			result = append(result, link)
			matchedLeft[left] = struct{}{}
			matchedRight[mostSimilarRight] = struct{}{}
		}
	}

	return result
}
