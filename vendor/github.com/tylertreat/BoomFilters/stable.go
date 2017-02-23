package boom

import (
	"hash"
	"hash/fnv"
	"math"
	"math/rand"
)

// StableBloomFilter implements a Stable Bloom Filter as described by Deng and
// Rafiei in Approximately Detecting Duplicates for Streaming Data using Stable
// Bloom Filters:
//
// http://webdocs.cs.ualberta.ca/~drafiei/papers/DupDet06Sigmod.pdf
//
// A Stable Bloom Filter (SBF) continuously evicts stale information so that it
// has room for more recent elements. Like traditional Bloom filters, an SBF
// has a non-zero probability of false positives, which is controlled by
// several parameters. Unlike the classic Bloom filter, an SBF has a tight
// upper bound on the rate of false positives while introducing a non-zero rate
// of false negatives. The false-positive rate of a classic Bloom filter
// eventually reaches 1, after which all queries result in a false positive.
// The stable-point property of an SBF means the false-positive rate
// asymptotically approaches a configurable fixed constant. A classic Bloom
// filter is actually a special case of SBF where the eviction rate is zero, so
// this package provides support for them as well.
//
// Stable Bloom Filters are useful for cases where the size of the data set
// isn't known a priori, which is a requirement for traditional Bloom filters,
// and memory is bounded.  For example, an SBF can be used to deduplicate
// events from an unbounded event stream with a specified upper bound on false
// positives and minimal false negatives.
type StableBloomFilter struct {
	cells       *Buckets    // filter data
	hash        hash.Hash64 // hash function (kernel for all k functions)
	m           uint        // number of cells
	p           uint        // number of cells to decrement
	k           uint        // number of hash functions
	max         uint8       // cell max value
	indexBuffer []uint      // buffer used to cache indices
}

// NewStableBloomFilter creates a new Stable Bloom Filter with m cells and d
// bits allocated per cell optimized for the target false-positive rate. Use
// NewDefaultStableFilter if you don't want to calculate d.
func NewStableBloomFilter(m uint, d uint8, fpRate float64) *StableBloomFilter {
	k := OptimalK(fpRate) / 2
	if k > m {
		k = m
	} else if k <= 0 {
		k = 1
	}

	cells := NewBuckets(m, d)

	return &StableBloomFilter{
		hash:        fnv.New64(),
		m:           m,
		k:           k,
		p:           optimalStableP(m, k, d, fpRate),
		max:         cells.MaxBucketValue(),
		cells:       cells,
		indexBuffer: make([]uint, k),
	}
}

// NewDefaultStableBloomFilter creates a new Stable Bloom Filter with m 1-bit
// cells and which is optimized for cases where there is no prior knowledge of
// the input data stream while maintaining an upper bound using the provided
// rate of false positives.
func NewDefaultStableBloomFilter(m uint, fpRate float64) *StableBloomFilter {
	return NewStableBloomFilter(m, 1, fpRate)
}

// NewUnstableBloomFilter creates a new special case of Stable Bloom Filter
// which is a traditional Bloom filter with m bits and an optimal number of
// hash functions for the target false-positive rate. Unlike the stable
// variant, data is not evicted and a cell contains a maximum of 1 hash value.
func NewUnstableBloomFilter(m uint, fpRate float64) *StableBloomFilter {
	var (
		cells = NewBuckets(m, 1)
		k     = OptimalK(fpRate)
	)

	return &StableBloomFilter{
		hash:        fnv.New64(),
		m:           m,
		k:           k,
		p:           0,
		max:         cells.MaxBucketValue(),
		cells:       cells,
		indexBuffer: make([]uint, k),
	}
}

// Cells returns the number of cells in the Stable Bloom Filter.
func (s *StableBloomFilter) Cells() uint {
	return s.m
}

// K returns the number of hash functions.
func (s *StableBloomFilter) K() uint {
	return s.k
}

// P returns the number of cells decremented on every add.
func (s *StableBloomFilter) P() uint {
	return s.p
}

// StablePoint returns the limit of the expected fraction of zeros in the
// Stable Bloom Filter when the number of iterations goes to infinity. When
// this limit is reached, the Stable Bloom Filter is considered stable.
func (s *StableBloomFilter) StablePoint() float64 {
	var (
		subDenom = float64(s.p) * (1/float64(s.k) - 1/float64(s.m))
		denom    = 1 + 1/subDenom
		base     = 1 / denom
	)

	return math.Pow(base, float64(s.max))
}

// FalsePositiveRate returns the upper bound on false positives when the filter
// has become stable.
func (s *StableBloomFilter) FalsePositiveRate() float64 {
	return math.Pow(1-s.StablePoint(), float64(s.k))
}

// Test will test for membership of the data and returns true if it is a
// member, false if not. This is a probabilistic test, meaning there is a
// non-zero probability of false positives and false negatives.
func (s *StableBloomFilter) Test(data []byte) bool {
	lower, upper := hashKernel(data, s.hash)

	// If any of the K cells are 0, then it's not a member.
	for i := uint(0); i < s.k; i++ {
		if s.cells.Get((uint(lower)+uint(upper)*i)%s.m) == 0 {
			return false
		}
	}

	return true
}

// Add will add the data to the Stable Bloom Filter. It returns the filter to
// allow for chaining.
func (s *StableBloomFilter) Add(data []byte) Filter {
	// Randomly decrement p cells to make room for new elements.
	s.decrement()

	lower, upper := hashKernel(data, s.hash)

	// Set the K cells to max.
	for i := uint(0); i < s.k; i++ {
		s.cells.Set((uint(lower)+uint(upper)*i)%s.m, s.max)
	}

	return s
}

// TestAndAdd is equivalent to calling Test followed by Add. It returns true if
// the data is a member, false if not.
func (s *StableBloomFilter) TestAndAdd(data []byte) bool {
	lower, upper := hashKernel(data, s.hash)
	member := true

	// If any of the K cells are 0, then it's not a member.
	for i := uint(0); i < s.k; i++ {
		s.indexBuffer[i] = (uint(lower) + uint(upper)*i) % s.m
		if s.cells.Get(s.indexBuffer[i]) == 0 {
			member = false
		}
	}

	// Randomly decrement p cells to make room for new elements.
	s.decrement()

	// Set the K cells to max.
	for _, idx := range s.indexBuffer {
		s.cells.Set(idx, s.max)
	}

	return member
}

// Reset restores the Stable Bloom Filter to its original state. It returns the
// filter to allow for chaining.
func (s *StableBloomFilter) Reset() *StableBloomFilter {
	s.cells.Reset()
	return s
}

// decrement will decrement a random cell and (p-1) adjacent cells by 1. This
// is faster than generating p random numbers. Although the processes of
// picking the p cells are not independent, each cell has a probability of p/m
// for being picked at each iteration, which means the properties still hold.
func (s *StableBloomFilter) decrement() {
	r := rand.Intn(int(s.m))
	for i := uint(0); i < s.p; i++ {
		idx := (r + int(i)) % int(s.m)
		s.cells.Increment(uint(idx), -1)
	}
}

// SetHash sets the hashing function used in the filter.
// For the effect on false positive rates see: https://github.com/tylertreat/BoomFilters/pull/1
func (s *StableBloomFilter) SetHash(h hash.Hash64) {
	s.hash = h
}

// optimalStableP returns the optimal number of cells to decrement, p, per
// iteration for the provided parameters of an SBF.
func optimalStableP(m, k uint, d uint8, fpRate float64) uint {
	var (
		max      = math.Pow(2, float64(d)) - 1
		subDenom = math.Pow(1-math.Pow(fpRate, 1/float64(k)), 1/max)
		denom    = (1/subDenom - 1) * (1/float64(k) - 1/float64(m))
	)

	p := uint(1 / denom)
	if p <= 0 {
		p = 1
	}

	return p
}
