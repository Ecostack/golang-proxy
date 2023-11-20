package util

import (
	"math/rand"
	"time"
)

func PickRandomString(items []string) string {
	if len(items) == 0 {
		return "" // Return empty string if the array is empty
	}
	rand.Seed(time.Now().UnixNano())     // Initialize the random number generator
	randomIndex := rand.Intn(len(items)) // Pick a random index
	return items[randomIndex]            // Return the string at the random index
}
