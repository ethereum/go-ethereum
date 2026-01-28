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
	"bytes"
	"crypto/sha256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

const (
	BasicDataLeafKey        = 0
	CodeHashLeafKey         = 1
	BasicDataCodeSizeOffset = 5
	BasicDataNonceOffset    = 8
	BasicDataBalanceOffset  = 16
)

var (
	zeroInt                             = uint256.NewInt(0)
	zeroHash                            = common.Hash{}
	verkleNodeWidthLog2                 = 8
	headerStorageOffset                 = uint256.NewInt(64)
	codeOffset                          = uint256.NewInt(128)
	codeStorageDelta                    = uint256.NewInt(0).Sub(codeOffset, headerStorageOffset)
	mainStorageOffsetLshVerkleNodeWidth = new(uint256.Int).Lsh(uint256.NewInt(1), 248-uint(verkleNodeWidthLog2))
	CodeOffset                          = uint256.NewInt(128)
	VerkleNodeWidth                     = uint256.NewInt(256)
	HeaderStorageOffset                 = uint256.NewInt(64)
	VerkleNodeWidthLog2                 = 8
)

func GetBinaryTreeKey(addr common.Address, key []byte) []byte {
	hasher := sha256.New()
	hasher.Write(zeroHash[:12])
	hasher.Write(addr[:])
	hasher.Write(key[:31])
	hasher.Write([]byte{0})
	k := hasher.Sum(nil)
	k[31] = key[31]
	return k
}

func GetBinaryTreeKeyBasicData(addr common.Address) []byte {
	var k [32]byte
	k[31] = BasicDataLeafKey
	return GetBinaryTreeKey(addr, k[:])
}

func GetBinaryTreeKeyCodeHash(addr common.Address) []byte {
	var k [32]byte
	k[31] = CodeHashLeafKey
	return GetBinaryTreeKey(addr, k[:])
}

func GetBinaryTreeKeyStorageSlot(address common.Address, key []byte) []byte {
	var k [32]byte

	// Case when the key belongs to the account header
	if bytes.Equal(key[:31], zeroHash[:31]) && key[31] < 64 {
		k[31] = 64 + key[31]
		return GetBinaryTreeKey(address, k[:])
	}

	// Set the main storage offset
	// note that the first 64 bytes of the main offset storage
	// are unreachable, which is consistent with the spec and
	// what verkle does.
	k[0] = 1 // 1 << 248
	copy(k[1:], key[:31])
	k[31] = key[31]

	return GetBinaryTreeKey(address, k[:])
}

func GetBinaryTreeKeyCodeChunk(address common.Address, chunknr *uint256.Int) []byte {
	chunkOffset := new(uint256.Int).Add(codeOffset, chunknr).Bytes()
	return GetBinaryTreeKey(address, chunkOffset)
}

func StorageIndex(storageKey []byte) (*uint256.Int, byte) {
	// If the storage slot is in the header, we need to add the header offset.
	var key uint256.Int
	key.SetBytes(storageKey)
	if key.Cmp(codeStorageDelta) < 0 {
		// This addition is always safe; it can't ever overflow since pos<codeStorageDelta.
		key.Add(headerStorageOffset, &key)

		// In this branch, the tree-index is zero since we're in the account header,
		// and the sub-index is the LSB of the modified storage key.
		return zeroInt, byte(key[0] & 0xFF)
	}
	// If the storage slot is in the main storage, we need to add the main storage offset.

	// The first MAIN_STORAGE_OFFSET group will see its
	// first 64 slots unreachable. This is either a typo in the
	// spec or intended to conserve the 256-u256
	// alignment. If we decide to ever access these 64
	// slots, uncomment this.
	// // Get the new offset since we now know that we are above 64.
	// pos.Sub(&pos, codeStorageDelta)
	// suffix := byte(pos[0] & 0xFF)
	suffix := storageKey[len(storageKey)-1]

	// We first divide by VerkleNodeWidth to create room to avoid an overflow next.
	key.Rsh(&key, uint(verkleNodeWidthLog2))

	// We add mainStorageOffset/VerkleNodeWidth which can't overflow.
	key.Add(&key, mainStorageOffsetLshVerkleNodeWidth)

	// The sub-index is the LSB of the original storage key, since mainStorageOffset
	// doesn't affect this byte, so we can avoid masks or shifts.
	return &key, suffix
}
