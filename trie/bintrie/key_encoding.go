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
	return getBinaryTreeKey(addr, key, false)
}

func getBinaryTreeKey(addr common.Address, offset []byte, overflow bool) []byte {
	hasher := sha256.New()
	hasher.Write(zeroHash[:12])
	hasher.Write(addr[:])
	var buf [32]byte
	// key is big endian, hashed value is little endian
	for i := range offset[:31] {
		buf[i] = offset[30-i]
	}
	if overflow {
		// Overflow detected when adding MAIN_STORAGE_OFFSET,
		// reporting it in the shifter 32 byte value.
		buf[31] = 1
	}
	hasher.Write(buf[:])
	k := hasher.Sum(nil)
	k[31] = offset[31]
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

func GetBinaryTreeKeyStorageSlot(address common.Address, slotnum []byte) []byte {
	var offset [32]byte

	// Case when the key belongs to the account header
	if bytes.Equal(slotnum[:31], zeroHash[:31]) && slotnum[31] < 64 {
		offset[31] = 64 + slotnum[31]
		return GetBinaryTreeKey(address, offset[:])
	}

	// Set the main storage offset offset = MAIN_STORAGE_OFFSET + slotnum
	//   * Note that MAIN_STORAGE_OFFSET is 1 << 248, so the number
	//     can overflow into a 33rd byte, but since the value is
	//     shifted by one byte in getBinaryTreeKey, this only takes
	//     note of the overflow, and the value will be added after
	//     the shift, in order to avoid allocating an extra byte.
	//   * Note that the first 64 bytes of the main offset storage
	//     are unreachable, which is consistent with the spec.
	//   * Note that `slotnum` is big-endian
	overflow := slotnum[0] == 255
	copy(offset[:], slotnum)
	offset[0] += 1 // 1 << 248, handle overflow out of band

	return getBinaryTreeKey(address, offset[:], overflow)
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
