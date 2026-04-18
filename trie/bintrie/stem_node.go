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

func (sn *StemNode) setValue(suffix byte, value []byte) {
	sn.values[suffix] = value
}

func (sn *StemNode) Hash() common.Hash {
	if !sn.mustRecompute {
		return sn.hash
	}

	var data [StemNodeWidth]common.Hash
	h := newSha256()
	defer returnSha256(h)

	for i, v := range sn.values {
		if v != nil {
			h.Reset()
			h.Write(v)
			h.Sum(data[i][:0])
		}
	}

	for level := 1; level <= 8; level++ {
		for i := range StemNodeWidth / (1 << level) {
			if data[i*2] == (common.Hash{}) && data[i*2+1] == (common.Hash{}) {
				data[i] = common.Hash{}
				continue
			}
			h.Reset()
			h.Write(data[i*2][:])
			h.Write(data[i*2+1][:])
			data[i] = common.Hash(h.Sum(nil))
		}
	}

	h.Reset()
	h.Write(sn.Stem[:])
	h.Write([]byte{0})
	h.Write(data[0][:])
	sn.hash = common.BytesToHash(h.Sum(nil))
	sn.mustRecompute = false
	return sn.hash
}

func (sn *StemNode) Key(i int) []byte {
	var ret [HashSize]byte
	copy(ret[:], sn.Stem[:])
	ret[StemSize] = byte(i)
	return ret[:]
}
