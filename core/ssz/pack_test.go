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
	"math/rand"
	"testing"
)

// naivePack is pack(values) transcribed from the spec.
func naivePack(data []byte) [][32]byte {
	var chunks [][32]byte
	for len(data) > 0 {
		var c [32]byte
		n := copy(c[:], data)
		chunks = append(chunks, c)
		data = data[n:]
	}
	return chunks
}

func TestPack(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, size := range []int{0, 1, 31, 32, 33, 63, 64, 65, 1000} {
		data := make([]byte, size)
		rng.Read(data)
		got, want := Pack(data), naivePack(data)
		if len(got) != (size+31)/32 {
			t.Errorf("size %d: %d chunks, want %d", size, len(got), (size+31)/32)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("size %d: chunk %d differs", size, i)
			}
		}
	}
}
