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
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

func keyToPath(depth int, key []byte) ([]byte, error) {
	if depth >= 31*8 {
		return nil, errors.New("node too deep")
	}
	keyLen := min(len(key), 31)
	ba := new(BitArray).SetBytes(uint8(keyLen*8), key[:keyLen])
	path := new(BitArray).MSBs(ba, uint8(depth+1))
	return path.ActiveBytes(), nil
}

// Invariant: dirty=false implies mustRecompute=false. Every mutation that
// invalidates the cached hash MUST also mark the blob for re-flush.
type InternalNode struct {
	left, right   nodeRef
	depth         uint8
	mustRecompute bool // hash is stale (cleared by Hash)
	dirty         bool // on-disk blob is stale (cleared by CollectNodes)
	hash          common.Hash
}
