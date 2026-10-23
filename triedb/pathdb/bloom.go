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

package pathdb

import (
	"encoding/binary"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
)

// Domain salts to keep the account, storage and trie node key spaces apart
// within the shared bloom filter, ensuring the identical raw bytes in the
// different domains won't alias to the same bloom key.
const (
	bloomAccountSalt = 0xff51afd7ed558ccd
	bloomStorageSalt = 0xc4ceb9fe1a85ec53
	bloomNodeSalt    = 0x9e3779b97f4a7c15
)

// bloomMix is the splitmix64 finalizer, scrambling the provided value into
// a uniformly distributed bloom key. The keccak-derived keys are uniform
// already, however, the trie node keys (owner + path) are constructed from
// the low entropy inputs and require the extra scrambling.
func bloomMix(x uint64) uint64 {
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	x ^= x >> 31
	return x
}

// bloomAccountKey derives the bloom key for the account with the specified
// account address hash.
func bloomAccountKey(hash common.Hash) uint64 {
	return bloomMix(binary.BigEndian.Uint64(hash[:8]) ^ bloomAccountSalt)
}

// bloomStorageSeed derives the seed shared by the storage slots of the
// account with the specified address hash.
func bloomStorageSeed(accountHash common.Hash) uint64 {
	x := binary.BigEndian.Uint64(accountHash[:8])
	return (x<<17 | x>>47) ^ bloomStorageSalt
}

// bloomSlotKey derives the bloom key for the storage slot with the specified
// slot key hash, under the given account seed.
func bloomSlotKey(seed uint64, storageHash common.Hash) uint64 {
	return bloomMix(seed ^ binary.BigEndian.Uint64(storageHash[:8]))
}

// bloomStorageKey derives the bloom key for the storage slot with the
// specified account address hash and slot key hash.
func bloomStorageKey(accountHash common.Hash, storageHash common.Hash) uint64 {
	return bloomSlotKey(bloomStorageSeed(accountHash), storageHash)
}

// bloomNodeSeed derives the seed shared by the trie nodes of the trie with
// the specified identifier.
func bloomNodeSeed(owner common.Hash) uint64 {
	return binary.BigEndian.Uint64(owner[:8]) ^ bloomNodeSalt
}

// bloomPathKey derives the bloom key for the trie node with the specified node
// path, under the given trie seed. The path is consumed 8 bytes at a time, the
// remainder is packed with its length to keep the different paths apart.
func bloomPathKey[P string | []byte](seed uint64, path P) uint64 {
	for len(path) >= 8 {
		seed = bloomMix(seed ^ bloomLoad64(path))
		path = path[8:]
	}
	tail := uint64(len(path)) << 56
	for i := 0; i < len(path); i++ {
		tail |= uint64(path[i]) << (8 * i)
	}
	return bloomMix(seed ^ tail)
}

// bloomLoad64 reads the leading 8 bytes of the path in little-endian order.
func bloomLoad64[P string | []byte](p P) uint64 {
	_ = p[7]
	return uint64(p[0]) | uint64(p[1])<<8 | uint64(p[2])<<16 | uint64(p[3])<<24 |
		uint64(p[4])<<32 | uint64(p[5])<<40 | uint64(p[6])<<48 | uint64(p[7])<<56
}

// bloomNodeKey derives the bloom key for the trie node with the specified
// trie identifier and node path.
func bloomNodeKey[P string | []byte](owner common.Hash, path P) uint64 {
	return bloomPathKey(bloomNodeSeed(owner), path)
}

// bloomFilter is a blocked bloom filter: the 3 bits of a key sit in one 512-bit
// block, so a probe touches a single cache line. It serves two purposes:
//
//   - The buffer-wide filter, short-circuiting the lookups that miss the buffer
//     entirely (the vast majority of the reads landing on the disk layer). It's
//     shared by the consecutive views and keeps growing while being probed: keys
//     are inserted with insertAtomic, before the content holding them is
//     published, so no false negative can be observed.
//
//   - The per-set filters, locating the set holding an entry without probing the
//     maps of the newer sets, which span megabytes and fall out of the CPU caches.
//     They're built with insert before the set is published and never modified.
//
// Keys are only ever inserted, never removed. Stale keys (e.g. left by a revert)
// merely increase the false positive rate, which is harmless for correctness.
type bloomFilter struct {
	blocks [][8]uint64
}

// newBloomFilter constructs a filter with at least the given number of bits.
func newBloomFilter(bits uint64) *bloomFilter {
	return &bloomFilter{blocks: make([][8]uint64, bits/512+1)}
}

// bloomBits returns the bloom bit count for the given buffer memory allowance.
// The mainnet buffer measured ~100-160 bits per distinct key with limit/4 bits
// (fp ~0.001%), limit/32 bits leaves 12-20 bits per key (fp ~0.3-1.2%). The
// count has a floor for the tiny buffers in tests; an allowance above the
// maxBufferSize (only in tests, the config is sanitized against it) is sized
// as the maximum.
func bloomBits(limit uint64) uint64 {
	return max(min(limit, maxBufferSize)/32, 1<<17)
}

// newBufferBloom constructs the buffer-wide filter for the given buffer memory
// allowance.
func newBufferBloom(limit uint64) *bloomFilter {
	return newBloomFilter(bloomBits(limit))
}

// newSetBloom constructs the filter over the provided keys, with 12 bits per key.
func newSetBloom(keys ...[]uint64) *bloomFilter {
	var n int
	for _, part := range keys {
		n += len(part)
	}
	b := newBloomFilter(uint64(n) * 12)
	for _, part := range keys {
		for _, key := range part {
			b.insert(key)
		}
	}
	return b
}

// block returns the block hosting the bits of the key, picked by the high 32
// bits of the key. The bits inside the block are picked by the low 27 bits.
func (b *bloomFilter) block(key uint64) *[8]uint64 {
	return &b.blocks[(key>>32)*uint64(len(b.blocks))>>32]
}

// insert adds the key into a filter that isn't probed concurrently.
func (b *bloomFilter) insert(key uint64) {
	blk := b.block(key)
	for i := range uint64(3) {
		bit := (key >> (9 * i)) & 511
		blk[bit/64] |= 1 << (bit % 64)
	}
}

// insertAtomic adds the key into a filter that may be probed concurrently. The
// callers must serialize the insertions themselves with respect to the content
// publication order.
func (b *bloomFilter) insertAtomic(key uint64) {
	blk := b.block(key)
	for i := range uint64(3) {
		bit := (key >> (9 * i)) & 511
		atomic.OrUint64(&blk[bit/64], 1<<(bit%64))
	}
}

// contains reports whether the key is possibly contained in the filter. False
// positives are possible, false negatives are not (for the keys inserted before
// the guarded content was published). A nil filter can't rule out any key.
func (b *bloomFilter) contains(key uint64) bool {
	if b == nil {
		return true
	}
	blk := b.block(key)
	for i := range uint64(3) {
		bit := (key >> (9 * i)) & 511
		if atomic.LoadUint64(&blk[bit/64])&(1<<(bit%64)) == 0 {
			return false
		}
	}
	return true
}

// bloomParts is the number of the partitions the bloom keys of a committed set
// are derived and inserted in, concurrently.
const bloomParts = 4

// appendBloomKeys appends the bloom keys of all the entries aggregated in the
// provided node set and state set to dst.
func appendBloomKeys(dst []uint64, nodes *nodeSet, states *stateSet) []uint64 {
	dst = slices.Grow(dst, bloomKeyCount(nodes, states))
	return appendBloomKeysPart(dst, nodes, states, 0, 1)
}

// appendBloomKeysPart appends the bloom keys of the given partition (out of
// parts) of the entries to dst. With multiple partitions, the partition 0 covers
// the account trie and the accounts, the others split the storage tries and the
// slots by the owner; a single partition covers all the entries.
func appendBloomKeysPart(dst []uint64, nodes *nodeSet, states *stateSet, part int, parts int) []uint64 {
	var (
		accounts = part == 0
		storage  = func(owner common.Hash) bool {
			return parts == 1 || part > 0 && int(owner[0])%(parts-1) == part-1
		}
	)
	if nodes != nil {
		if accounts {
			seed := bloomNodeSeed(common.Hash{})
			for path := range nodes.accountNodes {
				dst = append(dst, bloomPathKey(seed, path))
			}
		}
		for owner, subset := range nodes.storageNodes {
			if !storage(owner) {
				continue
			}
			seed := bloomNodeSeed(owner)
			for path := range subset {
				dst = append(dst, bloomPathKey(seed, path))
			}
		}
	}
	if states != nil {
		if accounts {
			for hash := range states.accountData {
				dst = append(dst, bloomAccountKey(hash))
			}
		}
		for accountHash, slots := range states.storageData {
			if !storage(accountHash) {
				continue
			}
			seed := bloomStorageSeed(accountHash)
			for storageHash := range slots {
				dst = append(dst, bloomSlotKey(seed, storageHash))
			}
		}
	}
	return dst
}

// bloomKeyCount returns the number of the entries aggregated in the provided
// node set and state set.
func bloomKeyCount(nodes *nodeSet, states *stateSet) int {
	var n int
	if nodes != nil {
		n += len(nodes.accountNodes)
		for _, subset := range nodes.storageNodes {
			n += len(subset)
		}
	}
	if states != nil {
		n += len(states.accountData)
		for _, slots := range states.storageData {
			n += len(slots)
		}
	}
	return n
}

// bloomKeyPool recycles the key slices staged between the key derivation and
// the construction of the filter of a committed set, one per partition.
var bloomKeyPool = sync.Pool{New: func() any { return new([bloomParts][]uint64) }}
