package linker

import (
	"slices"

	"github.com/antzucaro/matchr"
)

type ImplicitLink struct {
	Left        string
	Right       string
	Correlation float64
}

func CreateImplicitLinks(leftList, rightList []string) []ImplicitLink {
	var result []ImplicitLink
	var possible []ImplicitLink

	matchedLeft := make(map[string]struct{})
	matchedRight := make(map[string]struct{})

	for _, left := range leftList {
		_, matched := matchedLeft[left]
		if matched {
			continue
		}

		for _, right := range rightList {
			_, matched := matchedRight[right]
			if matched {
				continue
			}

			if left == right {
				link := ImplicitLink{
					Left:        left,
					Right:       right,
					Correlation: 1,
				}

				result = append(result, link)
				matchedLeft[left] = struct{}{}
				matchedRight[right] = struct{}{}
				break
			}

			similarity := matchr.JaroWinkler(left, right, false)
			possible = append(possible, ImplicitLink{
				Left:        left,
				Right:       right,
				Correlation: similarity,
			})
		}
	}

	slices.SortFunc(possible, func(a, b ImplicitLink) int {
		if a.Correlation < b.Correlation {
			return 1
		}
		if a.Correlation > b.Correlation {
			return -1
		}
		return 0
	})

	for _, link := range possible {
		_, matched := matchedLeft[link.Left]
		if matched {
			continue
		}
		_, matched = matchedRight[link.Right]
		if matched {
			continue
		}
		matchedLeft[link.Left] = struct{}{}
		matchedRight[link.Right] = struct{}{}
		result = append(result, link)
	}

	return result
}
