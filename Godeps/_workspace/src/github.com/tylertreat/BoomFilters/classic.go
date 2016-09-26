package boom

import (
	"hash"
	"hash/fnv"
	"math"
)

// BloomFilter implements a classic Bloom filter. A Bloom filter has a non-zero
// probability of false positives and a zero probability of false negatives.
type BloomFilter struct {
	buckets *Buckets    // filter data
	hash    hash.Hash64 // hash function (kernel for all k functions)
	m       uint        // filter size
	k       uint        // number of hash functions
	count   uint        // number of items added
}

// NewBloomFilter creates a new Bloom filter optimized to store n items with a
// specified target false-positive rate.
func NewBloomFilter(n uint, fpRate float64) *BloomFilter {
	m := OptimalM(n, fpRate)
	return &BloomFilter{
		buckets: NewBuckets(m, 1),
		hash:    fnv.New64(),
		m:       m,
		k:       OptimalK(fpRate),
	}
}

// Capacity returns the Bloom filter capacity, m.
func (b *BloomFilter) Capacity() uint {
	return b.m
}

// K returns the number of hash functions.
func (b *BloomFilter) K() uint {
	return b.k
}

// Count returns the number of items added to the filter.
func (b *BloomFilter) Count() uint {
	return b.count
}

// EstimatedFillRatio returns the current estimated ratio of set bits.
func (b *BloomFilter) EstimatedFillRatio() float64 {
	return 1 - math.Exp((-float64(b.count)*float64(b.k))/float64(b.m))
}

// FillRatio returns the ratio of set bits.
func (b *BloomFilter) FillRatio() float64 {
	sum := uint32(0)
	for i := uint(0); i < b.buckets.Count(); i++ {
		sum += b.buckets.Get(i)
	}
	return float64(sum) / float64(b.m)
}

// Test will test for membership of the data and returns true if it is a
// member, false if not. This is a probabilistic test, meaning there is a
// non-zero probability of false positives but a zero probability of false
// negatives.
func (b *BloomFilter) Test(data []byte) bool {
	lower, upper := hashKernel(data, b.hash)

	// If any of the K bits are not set, then it's not a member.
	for i := uint(0); i < b.k; i++ {
		if b.buckets.Get((uint(lower)+uint(upper)*i)%b.m) == 0 {
			return false
		}
	}

	return true
}

// Add will add the data to the Bloom filter. It returns the filter to allow
// for chaining.
func (b *BloomFilter) Add(data []byte) Filter {
	lower, upper := hashKernel(data, b.hash)

	// Set the K bits.
	for i := uint(0); i < b.k; i++ {
		b.buckets.Set((uint(lower)+uint(upper)*i)%b.m, 1)
	}

	b.count++
	return b
}

// TestAndAdd is equivalent to calling Test followed by Add. It returns true if
// the data is a member, false if not.
func (b *BloomFilter) TestAndAdd(data []byte) bool {
	lower, upper := hashKernel(data, b.hash)
	member := true

	// If any of the K bits are not set, then it's not a member.
	for i := uint(0); i < b.k; i++ {
		idx := (uint(lower) + uint(upper)*i) % b.m
		if b.buckets.Get(idx) == 0 {
			member = false
		}
		b.buckets.Set(idx, 1)
	}

	b.count++
	return member
}

// Reset restores the Bloom filter to its original state. It returns the filter
// to allow for chaining.
func (b *BloomFilter) Reset() *BloomFilter {
	b.buckets.Reset()
	return b
}

// SetHash sets the hashing function used in the filter.
// For the effect on false positive rates see: https://github.com/tylertreat/BoomFilters/pull/1
func (b *BloomFilter) SetHash(h hash.Hash64) {
	b.hash = h
}
