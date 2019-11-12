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

import (
	"math"

	"github.com/steakknife/hamming"
)

// PreciseFilledRatio is an exhaustive count # of 1's
func (f *Filter) PreciseFilledRatio() float64 {
	f.lock.RLock()
	defer f.lock.RUnlock()

	return float64(hamming.CountBitsUint64s(f.bits)) / float64(f.M())
}

// N is how many elements have been inserted
// (actually, how many Add()s have been performed?)
func (f *Filter) N() uint64 {
	f.lock.RLock()
	defer f.lock.RUnlock()

	return f.n
}

// FalsePosititveProbability is the upper-bound probability of false positives
//  (1 - exp(-k*(n+0.5)/(m-1))) ** k
func (f *Filter) FalsePosititveProbability() float64 {
	k := float64(f.K())
	n := float64(f.N())
	m := float64(f.M())
	return math.Pow(1.0-math.Exp(-k)*(n+0.5)/(m-1), k)
}
