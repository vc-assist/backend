package linker

import (
	jwd "github.com/jhvst/go-jaro-winkler-distance"
)

type ImplicitLink struct {
	Left        string
	Right       string
	Correlation float64
}

func CreateImplicitLinks(leftList, rightList []string) []ImplicitLink {
	var result []ImplicitLink
	matchedLeft := make(map[string]struct{})
	matchedRight := make(map[string]struct{})

leftExact:
	for _, left := range leftList {
		for _, right := range rightList {
			_, isMatchedRight := matchedRight[right]
			if isMatchedRight {
				continue
			}
			if left == right {
				result = append(result, ImplicitLink{
					Left:        left,
					Right:       right,
					Correlation: 1,
				})
				matchedLeft[left] = struct{}{}
				matchedRight[right] = struct{}{}
				continue leftExact
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

			similarity := jwd.Calculate(left, right)
			if similarity > mostSimilarity {
				mostSimilarity = similarity
				mostSimilarRight = right
			}
		}

		if mostSimilarity > 0 {
			result = append(result, ImplicitLink{
				Left:        left,
				Right:       mostSimilarRight,
				Correlation: mostSimilarity,
			})
			matchedLeft[left] = struct{}{}
			matchedRight[mostSimilarRight] = struct{}{}
		}
	}

	return result
}
