// Copyright 2025 go-ethereum Authors
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

package bintrie

import (
	"crypto/sha256"

	"github.com/ethereum/go-ethereum/common"
)

// StemNode holds up to 256 values sharing a 31-byte stem.
//
// Invariant: dirty=false implies mustRecompute=false. Every mutation that
// invalidates the cached hash MUST also mark the blob for re-flush.
type StemNode struct {
	Stem   [StemSize]byte
	values [StemNodeWidth][]byte // nil == slot absent

	depth uint8

	mustRecompute bool        // hash is stale (cleared by Hash)
	dirty         bool        // on-disk blob is stale (cleared by CollectNodes)
	hash          common.Hash // cached hash when mustRecompute == false
}

func (sn *StemNode) getValue(suffix byte) []byte {
	return sn.values[suffix]
}

func (sn *StemNode) hasValue(suffix byte) bool {
	return sn.values[suffix] != nil
}

// allValues returns the underlying slot array as a slice. nil entries mean
// absent. Callers must treat it as read-only.
func (sn *StemNode) allValues() [][]byte {
	return sn.values[:]
}

// setValue mutates a value slot and marks the stem for re-hash and
// re-flush. This is the only API for post-load value mutation; direct
// values[...] writes are reserved for the on-disk load path in
// decodeNode, which must leave mustRecompute/dirty at their loaded
// state.
func (sn *StemNode) setValue(suffix byte, value []byte) {
	sn.values[suffix] = value
	sn.mustRecompute = true
	sn.dirty = true
}

func (sn *StemNode) Hash() common.Hash {
	if !sn.mustRecompute {
		return sn.hash
	}

	// Use sha256.Sum256 (returns [32]byte by value) instead of a pooled
	// hash.Hash: feeding data[i][:0] into the interface method Sum forces
	// data to heap (escape analysis is conservative through interfaces).
	// Sum256 takes []byte and returns by value, so data stays on stack.
	var data [StemNodeWidth]common.Hash

	for i, v := range sn.values {
		if v != nil {
			data[i] = sha256.Sum256(v)
		}
	}

	var pair [2 * HashSize]byte
	for level := 1; level <= 8; level++ {
		for i := range StemNodeWidth / (1 << level) {
			if data[i*2] == (common.Hash{}) && data[i*2+1] == (common.Hash{}) {
				data[i] = common.Hash{}
				continue
			}
			copy(pair[:HashSize], data[i*2][:])
			copy(pair[HashSize:], data[i*2+1][:])
			data[i] = sha256.Sum256(pair[:])
		}
	}

	var final [StemSize + 1 + HashSize]byte
	copy(final[:StemSize], sn.Stem[:])
	final[StemSize] = 0
	copy(final[StemSize+1:], data[0][:])
	sn.hash = sha256.Sum256(final[:])
	sn.mustRecompute = false
	return sn.hash
}

func (sn *StemNode) Key(i int) []byte {
	var ret [HashSize]byte
	copy(ret[:], sn.Stem[:])
	ret[StemSize] = byte(i)
	return ret[:]
}
