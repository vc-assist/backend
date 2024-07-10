package linker

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestCreateImplicitLinks(t *testing.T) {
	testCases := []struct {
		left  []string
		right []string
		// if ImplicitLink.Correlation == 0
		// the test will not assert the correlation to be equal
		expected []ImplicitLink
	}{
		{
			left:  []string{"a", "b", "c"},
			right: []string{"a", "b"},
			expected: []ImplicitLink{
				{
					Left:        "a",
					Right:       "a",
					Correlation: 1,
				},
				{
					Left:        "b",
					Right:       "b",
					Correlation: 1,
				},
			},
		},
		{
			left:  []string{"foo", "bar", "baz"},
			right: []string{"foob", "bar", "barr"},
			expected: []ImplicitLink{
				{
					Left:        "bar",
					Right:       "bar",
					Correlation: 1,
				},
				{
					Left:  "baz",
					Right: "barr",
				},
				{
					Left:  "foo",
					Right: "foob",
				},
			},
		},
		{
			left:     []string{"foo", "bar", "baz"},
			right:    []string{},
			expected: nil,
		},
		{
			left:     []string{},
			right:    []string{},
			expected: nil,
		},
		{
			left:  []string{"foo", "bar", "baz"},
			right: []string{"baa"},
			expected: []ImplicitLink{
				{
					Left:  "bar",
					Right: "baa",
				},
			},
		},
	}

	for _, test := range testCases {
		links := CreateImplicitLinks(test.left, test.right)
		fmt.Println(links)
		diff := cmp.Diff(
			test.expected,
			links,
			cmpopts.SortSlices(func(a, b ImplicitLink) bool {
				return a.Left < b.Left
			}),
			cmpopts.IgnoreFields(ImplicitLink{}, "Correlation"),
		)
		if diff != "" {
			t.Fatal(diff)
		}
	}
}
