package boom

import (
	"hash"
	"hash/fnv"
)

// CountingBloomFilter implements a Counting Bloom Filter as described by Fan,
// Cao, Almeida, and Broder in Summary Cache: A Scalable Wide-Area Web Cache
// Sharing Protocol:
//
// http://pages.cs.wisc.edu/~jussara/papers/00ton.pdf
//
// A Counting Bloom Filter (CBF) provides a way to remove elements by using an
// array of n-bit buckets. When an element is added, the respective buckets are
// incremented. To remove an element, the respective buckets are decremented. A
// query checks that each of the respective buckets are non-zero. Because CBFs
// allow elements to be removed, they introduce a non-zero probability of false
// negatives in addition to the possibility of false positives.
//
// Counting Bloom Filters are useful for cases where elements are both added
// and removed from the data set. Since they use n-bit buckets, CBFs use
// roughly n-times more memory than traditional Bloom filters.
type CountingBloomFilter struct {
	buckets     *Buckets    // filter data
	hash        hash.Hash64 // hash function (kernel for all k functions)
	m           uint        // number of buckets
	k           uint        // number of hash functions
	count       uint        // number of items in the filter
	indexBuffer []uint      // buffer used to cache indices
}

// NewCountingBloomFilter creates a new Counting Bloom Filter optimized to
// store n items with a specified target false-positive rate and bucket size.
// If you don't know how many bits to use for buckets, use
// NewDefaultCountingBloomFilter for a sensible default.
func NewCountingBloomFilter(n uint, b uint8, fpRate float64) *CountingBloomFilter {
	var (
		m = OptimalM(n, fpRate)
		k = OptimalK(fpRate)
	)
	return &CountingBloomFilter{
		buckets:     NewBuckets(m, b),
		hash:        fnv.New64(),
		m:           m,
		k:           k,
		indexBuffer: make([]uint, k),
	}
}

// NewDefaultCountingBloomFilter creates a new Counting Bloom Filter optimized
// to store n items with a specified target false-positive rate. Buckets are
// allocated four bits.
func NewDefaultCountingBloomFilter(n uint, fpRate float64) *CountingBloomFilter {
	return NewCountingBloomFilter(n, 4, fpRate)
}

// Capacity returns the Bloom filter capacity, m.
func (c *CountingBloomFilter) Capacity() uint {
	return c.m
}

// K returns the number of hash functions.
func (c *CountingBloomFilter) K() uint {
	return c.k
}

// Count returns the number of items in the filter.
func (c *CountingBloomFilter) Count() uint {
	return c.count
}

// Test will test for membership of the data and returns true if it is a
// member, false if not. This is a probabilistic test, meaning there is a
// non-zero probability of false positives and false negatives.
func (c *CountingBloomFilter) Test(data []byte) bool {
	lower, upper := hashKernel(data, c.hash)

	// If any of the K bits are not set, then it's not a member.
	for i := uint(0); i < c.k; i++ {
		if c.buckets.Get((uint(lower)+uint(upper)*i)%c.m) == 0 {
			return false
		}
	}

	return true
}

// Add will add the data to the Bloom filter. It returns the filter to allow
// for chaining.
func (c *CountingBloomFilter) Add(data []byte) Filter {
	lower, upper := hashKernel(data, c.hash)

	// Set the K bits.
	for i := uint(0); i < c.k; i++ {
		c.buckets.Increment((uint(lower)+uint(upper)*i)%c.m, 1)
	}

	c.count++
	return c
}

// TestAndAdd is equivalent to calling Test followed by Add. It returns true if
// the data is a member, false if not.
func (c *CountingBloomFilter) TestAndAdd(data []byte) bool {
	lower, upper := hashKernel(data, c.hash)
	member := true

	// If any of the K bits are not set, then it's not a member.
	for i := uint(0); i < c.k; i++ {
		idx := (uint(lower) + uint(upper)*i) % c.m
		if c.buckets.Get(idx) == 0 {
			member = false
		}
		c.buckets.Increment(idx, 1)
	}

	c.count++
	return member
}

// TestAndRemove will test for membership of the data and remove it from the
// filter if it exists. Returns true if the data was a member, false if not.
func (c *CountingBloomFilter) TestAndRemove(data []byte) bool {
	lower, upper := hashKernel(data, c.hash)
	member := true

	// Set the K bits.
	for i := uint(0); i < c.k; i++ {
		c.indexBuffer[i] = (uint(lower) + uint(upper)*i) % c.m
		if c.buckets.Get(c.indexBuffer[i]) == 0 {
			member = false
		}
	}

	if member {
		for _, idx := range c.indexBuffer {
			c.buckets.Increment(idx, -1)
		}
		c.count--
	}

	return member
}

// Reset restores the Bloom filter to its original state. It returns the filter
// to allow for chaining.
func (c *CountingBloomFilter) Reset() *CountingBloomFilter {
	c.buckets.Reset()
	c.count = 0
	return c
}

// SetHash sets the hashing function used in the filter.
// For the effect on false positive rates see: https://github.com/tylertreat/BoomFilters/pull/1
func (c *CountingBloomFilter) SetHash(h hash.Hash64) {
	c.hash = h
}
