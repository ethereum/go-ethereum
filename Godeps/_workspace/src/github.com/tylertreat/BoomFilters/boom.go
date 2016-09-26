/*
Package boom implements probabilistic data structures for processing
continuous, unbounded data streams. This includes Stable Bloom Filters,
Scalable Bloom Filters, Counting Bloom Filters, Inverse Bloom Filters, several
variants of traditional Bloom filters, HyperLogLog, Count-Min Sketch, and
MinHash.

Classic Bloom filters generally require a priori knowledge of the data set
in order to allocate an appropriately sized bit array. This works well for
offline processing, but online processing typically involves unbounded data
streams. With enough data, a traditional Bloom filter "fills up", after
which it has a false-positive probability of 1.

Boom Filters are useful for situations where the size of the data set isn't
known ahead of time. For example, a Stable Bloom Filter can be used to
deduplicate events from an unbounded event stream with a specified upper
bound on false positives and minimal false negatives. Alternatively, an
Inverse Bloom Filter is ideal for deduplicating a stream where duplicate
events are relatively close together. This results in no false positives
and, depending on how close together duplicates are, a small probability of
false negatives. Scalable Bloom Filters place a tight upper bound on false
positives while avoiding false negatives but require allocating memory
proportional to the size of the data set. Counting Bloom Filters and Cuckoo
Filters are useful for cases which require adding and removing elements to and
from a set.

For large or unbounded data sets, calculating the exact cardinality is
impractical. HyperLogLog uses a fraction of the memory while providing an
accurate approximation. Similarly, Count-Min Sketch provides an efficient way
to estimate event frequency for data streams. TopK tracks the top-k most
frequent elements.

MinHash is a probabilistic algorithm to approximate the similarity between two
sets. This can be used to cluster or compare documents by splitting the corpus
into a bag of words.
*/
package boom

import (
	"encoding/binary"
	"hash"
	"math"
)

const fillRatio = 0.5

// Filter is a probabilistic data structure which is used to test the
// membership of an element in a set.
type Filter interface {
	// Test will test for membership of the data and returns true if it is a
	// member, false if not.
	Test([]byte) bool

	// Add will add the data to the Bloom filter. It returns the filter to
	// allow for chaining.
	Add([]byte) Filter

	// TestAndAdd is equivalent to calling Test followed by Add. It returns
	// true if the data is a member, false if not.
	TestAndAdd([]byte) bool
}

// OptimalM calculates the optimal Bloom filter size, m, based on the number of
// items and the desired rate of false positives.
func OptimalM(n uint, fpRate float64) uint {
	return uint(math.Ceil(float64(n) / ((math.Log(fillRatio) *
		math.Log(1-fillRatio)) / math.Abs(math.Log(fpRate)))))
}

// OptimalK calculates the optimal number of hash functions to use for a Bloom
// filter based on the desired rate of false positives.
func OptimalK(fpRate float64) uint {
	return uint(math.Ceil(math.Log2(1 / fpRate)))
}

// hashKernel returns the upper and lower base hash values from which the k
// hashes are derived.
func hashKernel(data []byte, hash hash.Hash64) (uint32, uint32) {
	hash.Write(data)
	sum := hash.Sum(nil)
	hash.Reset()
	return binary.BigEndian.Uint32(sum[4:8]), binary.BigEndian.Uint32(sum[0:4])
}
