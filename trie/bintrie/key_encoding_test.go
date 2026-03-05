// Copyright 2025 The go-ethereum Authors
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
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

func TestGetBinaryTreeKeyCodeChunkBoundaries(t *testing.T) {
	addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")

	for _, chunknr := range []uint64{0, 1, 127, 128, 255, 256, 257, 1024} {
		got := GetBinaryTreeKeyCodeChunk(addr, uint256.NewInt(chunknr))

		var offset [HashSize]byte
		binary.BigEndian.PutUint64(offset[24:], chunknr+128)
		want := GetBinaryTreeKey(addr, offset[:])

		if !bytes.Equal(got, want) {
			t.Fatalf("wrong code chunk key for chunk=%d: got=%x want=%x", chunknr, got, want)
		}
	}
}

func TestGetBinaryTreeKeyCodeChunkLargeIndex(t *testing.T) {
	addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")

	var chunknr uint256.Int
	chunknr.SetBytes(common.FromHex("0x0102030405060708090a0b0c0d0e0f10"))
	got := GetBinaryTreeKeyCodeChunk(addr, &chunknr)

	var offset uint256.Int
	offset.Add(codeOffset, &chunknr)
	offsetBytes := offset.Bytes32()
	want := GetBinaryTreeKey(addr, offsetBytes[:])

	if !bytes.Equal(got, want) {
		t.Fatalf("wrong code chunk key for large chunk index: got=%x want=%x", got, want)
	}
}
