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
	zeroHash   = common.Hash{}
	codeOffset = uint256.NewInt(128)
)

func GetBinaryTreeKey(addr common.Address, key []byte) []byte {
	hasher := sha256.New()
	hasher.Write(zeroHash[:12])
	hasher.Write(addr[:])
	hasher.Write(key[:31])
	k := hasher.Sum(nil)
	k[31] = key[31]
	return k
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
