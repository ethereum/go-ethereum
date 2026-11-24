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

package triedb

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie"
)

// assemblePartitionRoots mounts first-nibble subtries, emitting the root and
// any partition roots that survive path compression and are referenced by hash.
func assemblePartitionRoots(partitionBlobs *[numPartitions][]byte, onNode trie.OnTrieNode) (common.Hash, error) {
	var count, last int
	for i, blob := range partitionBlobs {
		if blob != nil {
			count++
			last = i
		}
	}
	switch count {
	case 0:
		return types.EmptyRootHash, nil
	case 1:
		child := partitionBlobs[last]
		hash, root, folded, err := trie.MountPartitionRoot(child, byte(last))
		if err != nil {
			return common.Hash{}, fmt.Errorf("mount partition %d: %w", last, err)
		}
		if !folded && len(child) >= common.HashLength {
			onNode([]byte{byte(last)}, crypto.Keccak256Hash(child), child)
		}
		onNode(nil, hash, root)
		return hash, nil
	}

	// Branch node
	var children [17][]byte
	for i, blob := range partitionBlobs {
		if blob == nil {
			continue
		}
		if len(blob) < common.HashLength {
			children[i] = blob
			continue
		}
		hash := crypto.Keccak256Hash(blob)
		onNode([]byte{byte(i)}, hash, blob)
		children[i] = hash[:]
	}
	root, hash, err := trie.AssembleBranch(children)
	if err != nil {
		return common.Hash{}, err
	}
	onNode(nil, hash, root)
	return hash, nil
}
