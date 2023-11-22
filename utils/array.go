package utils

import (
	"math/rand"
)

func PickRandomString(items []string) string {
	if len(items) == 0 {
		return "" // Return empty string if the array is empty
	}
	randomIndex := rand.Intn(len(items)) // Pick a random index
	return items[randomIndex]            // Return the string at the random index
}

func SelectWeighted[A any](items []A, weights []int) A {
	if len(items) != len(weights) {
		panic("Items and weights must be of the same length")
	}

	var total int
	cumulativeWeights := make([]int, len(items))
	for i, weight := range weights {
		total += weight
		cumulativeWeights[i] = total
	}

	// Generate a random number in the range of 0 to total
	r := rand.Intn(total)

	// Find where the random number falls in the cumulative weights array
	for i, cumWeight := range cumulativeWeights {
		if r < cumWeight {
			return items[i]
		}
	}

	panic("Should never reach here if weights are proper")
}
