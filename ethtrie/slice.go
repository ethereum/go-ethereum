package ethtrie

import "math"

// Helper function for comparing slices
func CompareIntSlice(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// Returns the amount of nibbles that match each other from 0 ...
func MatchingNibbleLength(a, b []int) int {
	var i, length = 0, int(math.Min(float64(len(a)), float64(len(b))))

	for i < length {
		if a[i] != b[i] {
			break
		}
		i++
	}

	return i
}
