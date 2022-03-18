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
	VerkleNodeWidth     = uint256.NewInt(256)
	codeStorageDelta    = uint256.NewInt(0).Sub(CodeOffset, HeaderStorageOffset)
)

func GetTreeKey(address []byte, treeIndex *uint256.Int, subIndex byte) []byte {
	var poly [256]fr.Element
	verkle.FromLEBytes(&poly[0], []byte{2, 64}) // little endian, 64 bytes
	verkle.FromLEBytes(&poly[1], address[:16])
	verkle.FromLEBytes(&poly[2], address[16:])
	// little-endian, 32-byte aligned treeIndex
	var index [32]byte
	for i, b := range treeIndex.Bytes() {
		index[len(treeIndex.Bytes())-1-i] = b
	}
	verkle.FromLEBytes(&poly[3], index[:16])
	verkle.FromLEBytes(&poly[4], index[16:])
	for i := 5; i < len(poly); i++ {
		verkle.CopyFr(&poly[i], &verkle.FrZero)
	}

	cfg, _ := verkle.GetConfig()
	ret := cfg.CommitToPoly(poly[:], 0)
	retb := ret.Bytes()
	retb[31] = subIndex
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
	pos := storageKey.Clone()
	if storageKey.Cmp(codeStorageDelta) < 0 {
		pos.Add(HeaderStorageOffset, storageKey)
	} else {
		pos.Add(MainStorageOffset, storageKey)
	}
	treeIndex := new(uint256.Int).Div(pos, VerkleNodeWidth)

	// calculate the sub_index, i.e. the index in the stem tree.
	// Because the modulus is 256, it's the last byte of treeIndex
	subIndexMod := new(uint256.Int).Mod(pos, VerkleNodeWidth).Bytes()
	var subIndex byte
	if len(subIndexMod) != 0 {
		// uint256 is broken into 4 little-endian quads,
		// each with native endianness. Extract the least
		// significant byte.
		subIndex = byte(subIndexMod[0] & 0xFF)
	}
	return GetTreeKey(address, treeIndex, subIndex)
}
