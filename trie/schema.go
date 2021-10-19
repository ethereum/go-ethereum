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

// NewNodePath converts an expanded trie path from nibble form into a compact
// version that can be sent over the network.
func NewNodePath(path []byte) NodePath {
	// If the hash is from the account trie, append a single item, if it
	// is from a storage trie, append a tuple. Note, the length 64 is
	// clashing between account leaf and storage root. It's fine though
	// because having a trie node at 64 depth means a hash collision was
	// found and we're long dead.
	if len(path) < 2*common.HashLength {
		return NodePath{hexToCompact(path)}
	}
	return NodePath{hexToKeybytes(path[:2*common.HashLength]), hexToCompact(path[2*common.HashLength:])}
}

// EncodeStorageKey combines the node owner and node path together to act as
// the unique database key for the trie node.
//
// The path part is encoded as the REVERSE-COMPACT format. It encodes all
// the nibbles into the hexary format and put the oddness flag in the end.
//
// The benefits of this key scheme are that:
// - it can group all the relevant trie nodes together to have data locality
//   in the database perspective.
// - it's space efficient. The keys obtained after encoding of adjacent nodes
//   have the same prefix which can cut down the real data size stored in the
//   underlying database
//
// The drawback of this scheme is: trie nodes can't be iterated in the key path
// ordering. Need to twist this format a bit if this property is needed.
func EncodeStorageKey(owner common.Hash, path []byte) []byte {
	var ret []byte
	if owner != (common.Hash{}) {
		ret = append(ret, owner.Bytes()...)
	}
	return append(ret, hexToSuffixCompact(path)...)
}

// DecodeStorageKey decodes the storage format node key and returns all the
// key components. The returned key is in hex nibbles.
func DecodeStorageKey(key []byte) (common.Hash, []byte) {
	if len(key) <= common.HashLength {
		return common.Hash{}, suffixCompactToHex(key)
	}
	return common.BytesToHash(key[:common.HashLength]), suffixCompactToHex(key[common.HashLength:])
}

// MaxStorageKeyLen returns the maximum storage key length. In practice,
// it's impossible to reach this length since valueNode is always embedded
// in the parent node. This function can be used to calculate the upper limit
// of encoded storage key.
func MaxStorageKeyLen() int {
	key := EncodeStorageKey(common.HexToHash("deadbeef"), keybytesToHex(common.Hash{}.Bytes()))
	return len(key)
}

// EncodeInternalKey appends the node hash in the end as a part of internal node
// key. This key format is used in memory for distinguishing different versions
// of trie nodes.
func EncodeInternalKey(storageKey []byte, hash common.Hash) []byte {
	if hash == (common.Hash{}) {
		panic("empty node hash")
	}
	return append(storageKey, hash.Bytes()...)
}

// EncodeInternalKeyWithPath encodes the passed key components into the internal
// key format.
func EncodeInternalKeyWithPath(owner common.Hash, path []byte, hash common.Hash) []byte {
	if hash == (common.Hash{}) {
		panic("empty node hash")
	}
	storage := EncodeStorageKey(owner, path)
	return EncodeInternalKey(storage, hash)
}

// DecodeInternalKey resolves the internalKey by removing the suffix hash.
// This key format is used as the database key for persisting trie node.
func DecodeInternalKey(internalKey []byte) ([]byte, common.Hash) {
	if len(internalKey) < common.HashLength {
		panic("invalid internal key")
	}
	return internalKey[:len(internalKey)-common.HashLength], common.BytesToHash(internalKey[len(internalKey)-common.HashLength:])
}
