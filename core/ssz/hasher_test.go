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
	"math/rand"
	"testing"
)

func randomChunks(rng *rand.Rand, n int) [][32]byte {
	chunks := make([][32]byte, n)
	for i := range chunks {
		rng.Read(chunks[i][:])
	}
	return chunks
}

func TestHashPair(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	chunks := randomChunks(rng, 2)
	var buf [64]byte
	copy(buf[:32], chunks[0][:])
	copy(buf[32:], chunks[1][:])
	if got, want := hashPair(chunks[0], chunks[1]), sha256.Sum256(buf[:]); got != want {
		t.Fatalf("hashPair = %x, want %x", got, want)
	}
}

func TestHashPairsInPlace(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	for _, n := range []int{1, 2, 3, 7, 64, 333} {
		chunks := randomChunks(rng, 2*n)
		want := make([][32]byte, n)
		hashPairs(want, chunks)
		for i := range want {
			if want[i] != hashPair(chunks[2*i], chunks[2*i+1]) {
				t.Fatalf("n=%d: digest %d differs from hashPair", n, i)
			}
		}
		// Folding a level onto its own front half must give the same result.
		hashPairs(chunks[:n], chunks)
		for i := range want {
			if chunks[i] != want[i] {
				t.Fatalf("n=%d: in-place digest %d differs", n, i)
			}
		}
	}
}

func TestHashPairsOddCountPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("hashPairs with an odd number of chunks did not panic")
		}
	}()
	hashPairs(make([][32]byte, 1), make([][32]byte, 3))
}
