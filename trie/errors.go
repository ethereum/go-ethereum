// Copyright 2015 The go-ethereum Authors
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

package trie

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// MissingNodeError is returned by the trie functions (TryGet, TryUpdate, TryDelete)
// in the case where a trie node is not present in the local database. Contains
// information necessary for retrieving the missing node through an ODR service.
//
// NodeHash is the hash of the missing node
//
// RootHash is the original root of the trie that contains the node
//
// PrefixLen is the nibble length of the key prefix that leads from the root to
// the missing node
//
// SuffixLen is the nibble length of the remaining part of the key that hints on
// which further nodes should also be retrieved (can be zero when there are no
// such hints in the error message)
type MissingNodeError struct {
	RootHash, NodeHash   common.Hash
	PrefixLen, SuffixLen int
}

func (err *MissingNodeError) Error() string {
	return fmt.Sprintf("Missing trie node %064x", err.NodeHash)
}
