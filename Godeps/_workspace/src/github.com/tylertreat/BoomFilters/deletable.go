package boom

import (
	"hash"
	"hash/fnv"
)

// DeletableBloomFilter implements a Deletable Bloom Filter as described by
// Rothenberg, Macapuna, Verdi, Magalhaes in The Deletable Bloom filter - A new
// member of the Bloom family:
//
// http://arxiv.org/pdf/1005.0352.pdf
//
// A Deletable Bloom Filter compactly stores information on collisions when
// inserting elements. This information is used to determine if elements are
// deletable. This design enables false-negative-free deletions at a fraction
// of the cost in memory consumption.
//
// Deletable Bloom Filters are useful for cases which require removing elements
// but cannot allow false negatives. This means they can be safely swapped in
// place of traditional Bloom filters.
type DeletableBloomFilter struct {
	buckets     *Buckets    // filter data
	collisions  *Buckets    // filter collision data
	hash        hash.Hash64 // hash function (kernel for all k functions)
	m           uint        // filter size
	regionSize  uint        // number of bits in a region
	k           uint        // number of hash functions
	count       uint        // number of items added
	indexBuffer []uint      // buffer used to cache indices
}

// NewDeletableBloomFilter creates a new DeletableBloomFilter optimized to
// store n items with a specified target false-positive rate. The r value
// determines the number of bits to use to store collision information. This
// controls the deletability of an element. Refer to the paper for selecting an
// optimal value.
func NewDeletableBloomFilter(n, r uint, fpRate float64) *DeletableBloomFilter {
	var (
		m = OptimalM(n, fpRate)
		k = OptimalK(fpRate)
	)
	return &DeletableBloomFilter{
		buckets:     NewBuckets(m-r, 1),
		collisions:  NewBuckets(r, 1),
		hash:        fnv.New64(),
		m:           m - r,
		regionSize:  (m - r) / r,
		k:           k,
		indexBuffer: make([]uint, k),
	}
}

// Capacity returns the Bloom filter capacity, m.
func (d *DeletableBloomFilter) Capacity() uint {
	return d.m
}

// K returns the number of hash functions.
func (d *DeletableBloomFilter) K() uint {
	return d.k
}

// Count returns the number of items added to the filter.
func (d *DeletableBloomFilter) Count() uint {
	return d.count
}

// Test will test for membership of the data and returns true if it is a
// member, false if not. This is a probabilistic test, meaning there is a
// non-zero probability of false positives but a zero probability of false
// negatives.
func (d *DeletableBloomFilter) Test(data []byte) bool {
	lower, upper := hashKernel(data, d.hash)

	// If any of the K bits are not set, then it's not a member.
	for i := uint(0); i < d.k; i++ {
		if d.buckets.Get((uint(lower)+uint(upper)*i)%d.m) == 0 {
			return false
		}
	}

	return true
}

// Add will add the data to the Bloom filter. It returns the filter to allow
// for chaining.
func (d *DeletableBloomFilter) Add(data []byte) Filter {
	lower, upper := hashKernel(data, d.hash)

	// Set the K bits.
	for i := uint(0); i < d.k; i++ {
		idx := (uint(lower) + uint(upper)*i) % d.m
		if d.buckets.Get(idx) != 0 {
			// Collision, set corresponding region bit.
			d.collisions.Set(idx/d.regionSize, 1)
		} else {
			d.buckets.Set(idx, 1)
		}
	}

	d.count++
	return d
}

// TestAndAdd is equivalent to calling Test followed by Add. It returns true if
// the data is a member, false if not.
func (d *DeletableBloomFilter) TestAndAdd(data []byte) bool {
	lower, upper := hashKernel(data, d.hash)
	member := true

	// If any of the K bits are not set, then it's not a member.
	for i := uint(0); i < d.k; i++ {
		idx := (uint(lower) + uint(upper)*i) % d.m
		if d.buckets.Get(idx) == 0 {
			member = false
		} else {
			// Collision, set corresponding region bit.
			d.collisions.Set(idx/d.regionSize, 1)
		}
		d.buckets.Set(idx, 1)
	}

	d.count++
	return member
}

// TestAndRemove will test for membership of the data and remove it from the
// filter if it exists. Returns true if the data was a member, false if not.
func (d *DeletableBloomFilter) TestAndRemove(data []byte) bool {
	lower, upper := hashKernel(data, d.hash)
	member := true

	// Set the K bits.
	for i := uint(0); i < d.k; i++ {
		d.indexBuffer[i] = (uint(lower) + uint(upper)*i) % d.m
		if d.buckets.Get(d.indexBuffer[i]) == 0 {
			member = false
		}
	}

	if member {
		for _, idx := range d.indexBuffer {
			if d.collisions.Get(idx/d.regionSize) == 0 {
				// Clear only bits located in collision-free zones.
				d.buckets.Set(idx, 0)
			}
		}
		d.count--
	}

	return member
}

// Reset restores the Bloom filter to its original state. It returns the filter
// to allow for chaining.
func (d *DeletableBloomFilter) Reset() *DeletableBloomFilter {
	d.buckets.Reset()
	d.collisions.Reset()
	d.count = 0
	return d
}

// SetHash sets the hashing function used in the filter.
// For the effect on false positive rates see: https://github.com/tylertreat/BoomFilters/pull/1
func (d *DeletableBloomFilter) SetHash(h hash.Hash64) {
	d.hash = h
}
