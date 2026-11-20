// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package legacypool

import (
	"maps"
	"math/bits"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

const (
	// feeBucketBits is how many bits of a fee cap below its leading one pick
	// its bucket: 32 buckets per doubling, each 2-3% wide.
	feeBucketBits = 5

	// feeBucketCount is the number of buckets covering every uint256 fee cap.
	feeBucketCount = (257 - feeBucketBits) << feeBucketBits
)

// feeBucket returns the bucket of a fee cap. Buckets are ordered as the fee
// caps in them: below 64 each value has its own, above it the position of the
// leading one and the five bits after it pick one.
func feeBucket(feeCap *uint256.Int) uint16 {
	n := feeCap.BitLen()
	if n <= feeBucketBits+1 {
		return uint16(feeCap.Uint64())
	}
	shift := uint(n - 1 - feeBucketBits)
	var top uint256.Int
	top.Rsh(feeCap, shift) // in [1<<feeBucketBits, 2<<feeBucketBits)
	return uint16(shift)<<feeBucketBits + uint16(top.Uint64())
}

// headIndex groups the pending accounts by their first transaction's fee cap,
// so that Pending can skip the accounts whose first transaction can't pay the
// base fee. The pending lists keep it current under the pool's lock.
type headIndex struct {
	buckets map[uint16]*headBucket
	used    [feeBucketCount / 64]uint64 // bit k%64 of used[k/64]: bucket k is not empty
	bucket  map[common.Address]uint16   // each indexed account's bucket
}

type headEntry struct {
	list   *list
	feeCap uint256.Int // of the list's first transaction
}

// headBucketMinPeak is the size below which a bucket's map is never rebuilt.
const headBucketMinPeak = 64

// headBucket is the accounts of one fee cap bucket. A Go map keeps its size as
// entries leave, so a bucket's map is rebuilt once it holds less than a quarter
// of its peak: senders moving through the buckets would otherwise leave large,
// almost empty maps behind, which cost memory and slow Pending's walk.
type headBucket struct {
	entries map[common.Address]headEntry
	peak    int // the most entries since the map was made
}

func newHeadIndex() *headIndex {
	return &headIndex{
		buckets: make(map[uint16]*headBucket),
		bucket:  make(map[common.Address]uint16),
	}
}

// update indexes addr by the first transaction of its pending list l, or
// drops it if l is empty.
func (idx *headIndex) update(addr common.Address, l *list) {
	old, indexed := idx.bucket[addr]
	if l.txs.index.Len() == 0 {
		if indexed {
			idx.remove(addr, old)
		}
		return
	}
	entry := headEntry{list: l}
	entry.feeCap, _ = l.txs.first().GasFeeCapUint256() // the pool admits none that overflow
	k := feeBucket(&entry.feeCap)
	if indexed && old != k {
		idx.remove(addr, old)
	}
	bucket := idx.buckets[k]
	if bucket == nil {
		bucket = &headBucket{entries: make(map[common.Address]headEntry)}
		idx.buckets[k] = bucket
		idx.used[k/64] |= 1 << (k % 64)
	}
	bucket.entries[addr] = entry
	bucket.peak = max(bucket.peak, len(bucket.entries))
	idx.bucket[addr] = k
}

func (idx *headIndex) remove(addr common.Address, k uint16) {
	bucket := idx.buckets[k]
	delete(bucket.entries, addr)
	switch n := len(bucket.entries); {
	case n == 0:
		delete(idx.buckets, k)
		idx.used[k/64] &^= 1 << (k % 64)
	case bucket.peak >= headBucketMinPeak && n < bucket.peak/4:
		// A copy, not maps.Clone, which keeps the map's size.
		entries := make(map[common.Address]headEntry, n)
		maps.Copy(entries, bucket.entries)
		bucket.entries, bucket.peak = entries, n
	}
	delete(idx.bucket, addr)
}

// atLeast calls f on every account whose first pending transaction has a fee
// cap of at least feeCap.
func (idx *headIndex) atLeast(feeCap *uint256.Int, f func(common.Address, *list)) {
	lowest := feeBucket(feeCap)
	for w := len(idx.used) - 1; w >= int(lowest/64); w-- {
		used := idx.used[w]
		if w == int(lowest/64) {
			used &= ^uint64(0) << (lowest % 64)
		}
		for used != 0 {
			b := 63 - bits.LeadingZeros64(used)
			used &^= 1 << b
			k := uint16(w*64 + b)
			for addr, entry := range idx.buckets[k].entries {
				if k == lowest && entry.feeCap.Lt(feeCap) {
					continue
				}
				f(addr, entry.list)
			}
		}
	}
}
