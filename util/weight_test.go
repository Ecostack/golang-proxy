package util

import "testing"

func TestSelectWeighted(t *testing.T) {
	items := []string{"a", "b", "c", "d"}
	weights := []int{100, 200, 300, 400}
	selections := make(map[string]int)
	for i := 0; i < 1000; i++ {
		selections[PickRandomString(items)]++
	}
	for i := 0; i < 1000; i++ {
		selections[SelectWeighted(items, weights)]++
	}
	for _, item := range items {
		if selections[item] < 200 {
			t.Errorf("Item %s was selected less than 200 times", item)
		}
	}
}
