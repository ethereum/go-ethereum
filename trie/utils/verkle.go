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
	"crypto/sha256"

	"github.com/ethereum/go-ethereum/common"
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
	codeStorageDelta    = uint256.NewInt(0).Sub(HeaderStorageOffset, CodeOffset)
)

func GetTreeKey(address common.Address, treeIndex *uint256.Int, subIndex byte) []byte {
	digest := sha256.New()
	digest.Write(address[:])
	treeIndexBytes := treeIndex.Bytes()
	var payload [32]byte
	copy(payload[:len(treeIndexBytes)], treeIndexBytes)
	digest.Write(payload[:])
	h := digest.Sum(nil)
	h[31] = byte(subIndex)
	return h
}

func GetTreeKeyVersion(address common.Address) []byte {
	return GetTreeKey(address, zero, VersionLeafKey)
}

func GetTreeKeyBalance(address common.Address) []byte {
	return GetTreeKey(address, zero, BalanceLeafKey)
}

func GetTreeKeyNonce(address common.Address) []byte {
	return GetTreeKey(address, zero, NonceLeafKey)
}

func GetTreeKeyCodeKeccak(address common.Address) []byte {
	return GetTreeKey(address, zero, CodeKeccakLeafKey)
}

func GetTreeKeyCodeSize(address common.Address) []byte {
	return GetTreeKey(address, zero, CodeSizeLeafKey)
}

func GetTreeKeyCodeChunk(address common.Address, chunk *uint256.Int) []byte {
	chunkOffset := new(uint256.Int).Add(CodeOffset, chunk)
	treeIndex := new(uint256.Int).Div(chunkOffset, VerkleNodeWidth)
	subIndex := new(uint256.Int).Mod(chunkOffset, VerkleNodeWidth).Bytes()[0]
	return GetTreeKey(address, treeIndex, subIndex)
}

func GetTreeKeyStorageSlot(address common.Address, storageKey *uint256.Int) []byte {
	if storageKey.Cmp(codeStorageDelta) < 0 {
		storageKey.Add(HeaderStorageOffset, storageKey)
	} else {
		storageKey.Add(MainStorageOffset, storageKey)
	}
	treeIndex := new(uint256.Int).Div(storageKey, VerkleNodeWidth)
	subIndex := new(uint256.Int).Mod(storageKey, VerkleNodeWidth).Bytes()[0]
	return GetTreeKey(address, treeIndex, subIndex)
}
