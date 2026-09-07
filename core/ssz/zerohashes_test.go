// Copyright 2026 The go-ethereum Authors
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

package ssz

import (
	"crypto/sha256"
	"testing"
)

func TestZeroHashes(t *testing.T) {
	var want [32]byte
	for d := range zeroHashes {
		if zeroHashes[d] != want {
			t.Fatalf("zeroHashes[%d] differs from recomputation", d)
		}
		var buf [64]byte
		copy(buf[:32], want[:])
		copy(buf[32:], want[:])
		want = sha256.Sum256(buf[:])
	}
}
