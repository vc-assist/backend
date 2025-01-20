package testutil

import (
	"fmt"
	"math/rand"
)

// RandomSwitch returns a function that will output various integers at different weights.
//
// Ex. RandomSwitch(2, 3, 5) will return a function that will output:
//   - `0` 20% of the time
//   - `1` 30% of the time
//   - `2` 50% of the time
func RandomSwitch(weights ...int) func(rndm *rand.Rand) int {
	if len(weights) == 0 {
		panic("a random switch must have at least 1 probability")
	}

	var sum int
	for _, p := range weights {
		if p == 0 {
			panic("cannot have weight that is 0")
		}
		sum += p
	}

	return func(rndm *rand.Rand) int {
		value := rndm.Intn(sum)

		threshold := 0
		for i := 0; i < len(weights); i++ {
			threshold += weights[i]
			if value < threshold {
				return i
			}
		}

		panic(fmt.Sprintf("random value generated was out of bounds: %d", value))
	}
}

// RandomString generates a random lowercase string given the pseudo random source.
func RandomString(rndm *rand.Rand, length int) string {
	str := make([]rune, length)
	for i := range length {
		str[i] = 'a' + rune(rndm.Intn(26))
	}
	return string(str)
}
