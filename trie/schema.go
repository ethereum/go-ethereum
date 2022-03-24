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

import "github.com/ethereum/go-ethereum/common"

// EncodeStorageKey combines the node owner and node path together to act as
// the unique database key for the trie node.
//
// The path part is encoded as the SUFFIX-COMPACT format. It encodes all
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
// in the parent node. This function can be used to calculate the upper
// bound length of encoded storage key.
func MaxStorageKeyLen() int {
	key := EncodeStorageKey(common.HexToHash("deadbeef"), keybytesToHex(common.Hash{}.Bytes()))
	return len(key)
}

// EncodeKeyWithHash appends the node hash at the end as a part of node key.
// This key can uniquely represent a node in trie.
func EncodeKeyWithHash(storageKey []byte, hash common.Hash) string {
	if hash == (common.Hash{}) {
		panic("empty node hash")
	}
	return string(append(storageKey, hash.Bytes()...))
}

// DecodeKeyWithHash resolves the unique key by splitting the key into two
// parts: the storage key used for persisting node and the node hash.
func DecodeKeyWithHash(key []byte) ([]byte, common.Hash) {
	if len(key) < common.HashLength {
		panic("invalid embedded key")
	}
	return key[:len(key)-common.HashLength], common.BytesToHash(key[len(key)-common.HashLength:])
}
