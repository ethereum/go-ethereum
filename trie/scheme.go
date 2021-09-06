// Copyright 2021 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/common"
)

// NodePath is a path tuple identifying a particular trie node either in a single
// trie (account) or a layered trie (account -> storage).
//
// Content wise the tuple either has 1 element if it addresses a node in a single
// trie or 2 elements if it addresses a node in a stacked trie.
//
// To support aiming arbitrary trie nodes, the path needs to support odd nibble
// lengths. To avoid transferring expanded hex form over the network, the last
// part of the tuple (which needs to index into the middle of a trie) is compact
// encoded. In case of a 2-tuple, the first item is always 32 bytes so that is
// simple binary encoded.
//
// Examples:
//   - Path 0x9  -> {0x19}
//   - Path 0x99 -> {0x0099}
//   - Path 0x01234567890123456789012345678901012345678901234567890123456789019  -> {0x0123456789012345678901234567890101234567890123456789012345678901, 0x19}
//   - Path 0x012345678901234567890123456789010123456789012345678901234567890199 -> {0x0123456789012345678901234567890101234567890123456789012345678901, 0x0099}
type NodePath [][]byte

// newNodePath converts an expanded trie path from nibble form into a compact
// version that can be sent over the network.
func newNodePath(path []byte) NodePath {
	// If the hash is from the account trie, append a single item, if it
	// is from the a storage trie, append a tuple. Note, the length 64 is
	// clashing between account leaf and storage root. It's fine though
	// because having a trie node at 64 depth means a hash collision was
	// found and we're long dead.
	if len(path) < 64 {
		return NodePath{hexToCompact(path)}
	}
	return NodePath{HexToKeybytes(path[:64]), hexToCompact(path[64:])}
}

func encodeNodeKey(owner common.Hash, path []byte) []byte {
	var ret []byte
	if owner != (common.Hash{}) {
		ret = append(ret, owner.Bytes()...)
	}
	return append(ret, hexToCompact(path)...)
}

func decodeNodeKey(key []byte) (common.Hash, []byte) {
	if len(key) <= common.HashLength {
		return common.Hash{}, compactToHex(key)
	}
	return common.BytesToHash(key[:common.HashLength]), compactToHex(key[common.HashLength:])
}

func TrieRootKey(owner common.Hash) []byte {
	return encodeNodeKey(owner, nil)
}
