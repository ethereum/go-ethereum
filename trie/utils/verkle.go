// Copyright 2021 go-ethereum Authors
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

package utils

import (
	"github.com/crate-crypto/go-ipa/bandersnatch/fr"
	"github.com/crate-crypto/go-ipa/ipa"
	"github.com/gballet/go-verkle"

	"github.com/holiman/uint256"
)

const (
	VersionLeafKey    = 0
	BalanceLeafKey    = 1
	NonceLeafKey      = 2
	CodeKeccakLeafKey = 3
	CodeSizeLeafKey   = 4
)

var (
	zero                = uint256.NewInt(0)
	HeaderStorageOffset = uint256.NewInt(64)
	CodeOffset          = uint256.NewInt(128)
	MainStorageOffset   = new(uint256.Int).Lsh(uint256.NewInt(256), 31)
	VerkleNodeWidth     = uint256.NewInt(8)
	codeStorageDelta    = uint256.NewInt(0).Sub(CodeOffset, HeaderStorageOffset)
)

func GetTreeKey(address []byte, treeIndex *uint256.Int, subIndex byte) []byte {
	var poly []fr.Element
	verkle.FromLEBytes(&poly[0], []byte{2, 63})
	verkle.FromLEBytes(&poly[1], address[:16])
	verkle.FromLEBytes(&poly[2], address[16:])
	verkle.FromLEBytes(&poly[3], treeIndex.Bytes()[:16])
	verkle.FromLEBytes(&poly[4], treeIndex.Bytes()[16:])
	for i := 5; i < len(poly); i++ {
		verkle.CopyFr(&poly[i], &verkle.FrZero)
	}

	ret := ipa.NewIPASettings().Commit(poly)
	retb := ret.Bytes()
	return retb[:]

}

func GetTreeKeyAccountLeaf(address []byte, leaf byte) []byte {
	return GetTreeKey(address, zero, leaf)
}

func GetTreeKeyVersion(address []byte) []byte {
	return GetTreeKey(address, zero, VersionLeafKey)
}

func GetTreeKeyBalance(address []byte) []byte {
	return GetTreeKey(address, zero, BalanceLeafKey)
}

func GetTreeKeyNonce(address []byte) []byte {
	return GetTreeKey(address, zero, NonceLeafKey)
}

func GetTreeKeyCodeKeccak(address []byte) []byte {
	return GetTreeKey(address, zero, CodeKeccakLeafKey)
}

func GetTreeKeyCodeSize(address []byte) []byte {
	return GetTreeKey(address, zero, CodeSizeLeafKey)
}

func GetTreeKeyCodeChunk(address []byte, chunk *uint256.Int) []byte {
	chunkOffset := new(uint256.Int).Add(CodeOffset, chunk)
	treeIndex := new(uint256.Int).Div(chunkOffset, VerkleNodeWidth)
	subIndexMod := new(uint256.Int).Mod(chunkOffset, VerkleNodeWidth).Bytes()
	var subIndex byte
	if len(subIndexMod) != 0 {
		subIndex = subIndexMod[0]
	}
	return GetTreeKey(address, treeIndex, subIndex)
}

func GetTreeKeyStorageSlot(address []byte, storageKey *uint256.Int) []byte {
	treeIndex := storageKey.Clone()
	if storageKey.Cmp(codeStorageDelta) < 0 {
		treeIndex.Add(HeaderStorageOffset, storageKey)
	} else {
		treeIndex.Add(MainStorageOffset, storageKey)
	}
	treeIndex.Div(treeIndex, VerkleNodeWidth)

	// calculate the sub_index, i.e. the index in the stem tree.
	// Because the modulus is 256, it's the last byte of treeIndex
	subIndexMod := treeIndex.Bytes()
	var subIndex byte
	if len(subIndexMod) != 0 {
		// Get the last byte, as uint256.Int is big-endian
		subIndex = subIndexMod[len(subIndexMod)-1]
	}
	return GetTreeKey(address, treeIndex, subIndex)
}
