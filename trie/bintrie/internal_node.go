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
	path := make([]byte, 0, depth+1)
	for i := range depth + 1 {
		bit := key[i/8] >> (7 - (i % 8)) & 1
		path = append(path, bit)
	}
	return path, nil
}

// makeKeyPath is a simplified version of keyToPath that doesn't return an error.
func makeKeyPath(depth int, key []byte) []byte {
	path := make([]byte, 0, depth+1)
	for i := range depth + 1 {
		bit := key[i/8] >> (7 - (i % 8)) & 1
		path = append(path, bit)
	}
	return path
}

// InternalNode is a binary trie internal node.
type InternalNode struct {
	left, right   NodeRef
	depth         uint8
	mustRecompute bool        // true if the hash needs to be recomputed
	hash          common.Hash // cached hash when mustRecompute == false
}
