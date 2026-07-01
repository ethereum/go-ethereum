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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// TestGetBinaryTreeKeyCodeChunkZero is a regression test: chunknr=0 encodes
// codeOffset (128) to a single byte, which caused getBinaryTreeKey to panic on
// offset[:31] before the offset was zero-padded to 32 bytes.
func TestGetBinaryTreeKeyCodeChunkZero(t *testing.T) {
	key := GetBinaryTreeKeyCodeChunk(common.Address{}, uint256.NewInt(0))
	if len(key) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(key))
	}
}
