// Package bloomfilter is face-meltingly fast, thread-safe,
// marshalable, unionable, probability- and
// optimal-size-calculating Bloom filter in go
//
// https://github.com/steakknife/bloomfilter
//
// Copyright © 2014, 2015, 2018 Barry Allard
//
// MIT license
//
package bloomfilter

import "math"

const gigabitsPerGiB float64 = 8.0 * 1024 * 1024 * 1024

// OptimalK calculates the optimal k value for creating a new Bloom filter
// maxn is the maximum anticipated number of elements
func OptimalK(m, maxN uint64) uint64 {
	return uint64(math.Ceil(float64(m) * math.Ln2 / float64(maxN)))
}

// OptimalM calculates the optimal m value for creating a new Bloom filter
// p is the desired false positive probability
// optimal m = ceiling( - n * ln(p) / ln(2)**2 )
func OptimalM(maxN uint64, p float64) uint64 {
	return uint64(math.Ceil(-float64(maxN) * math.Log(p) / (math.Ln2 * math.Ln2)))
}
