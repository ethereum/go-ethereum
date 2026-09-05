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
	"strconv"
	"testing"
)

// hashPairsGeneric is the reference for hashPairs: the same pairwise fold
// over crypto/sha256, one node at a time.
func hashPairsGeneric(digests, chunks [][32]byte) {
	for i := range len(chunks) / 2 {
		digests[i] = hashPair(chunks[2*i], chunks[2*i+1])
	}
}

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
		hashPairsGeneric(want, chunks)
		got := make([][32]byte, n)
		hashPairs(got, chunks)
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("n=%d: digest %d differs from generic", n, i)
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

func TestHashPairsShortOutputPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("hashPairs with a too-short digests slice did not panic")
		}
	}()
	hashPairs(make([][32]byte, 1), make([][32]byte, 4))
}

// FuzzHashPairs checks that the batched backend and the generic fold agree
// bit for bit, both into a separate buffer and in place, for every level
// shape the fuzzer can produce.
func FuzzHashPairs(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 64))
	f.Add(make([]byte, 64*7))
	f.Add(make([]byte, 64*100))
	f.Fuzz(func(t *testing.T, data []byte) {
		chunks := Pack(data)
		chunks = chunks[:len(chunks)/2*2]
		n := len(chunks) / 2
		want := make([][32]byte, n)
		hashPairsGeneric(want, chunks)
		got := make([][32]byte, n)
		hashPairs(got, chunks)
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%d chunks: digest %d differs", len(chunks), i)
			}
		}
		hashPairs(chunks[:n], chunks)
		for i := range want {
			if chunks[i] != want[i] {
				t.Fatalf("%d chunks: in-place digest %d differs", len(chunks), i)
			}
		}
	})
}

func BenchmarkHashPairs(b *testing.B) {
	rng := rand.New(rand.NewSource(4))
	for _, n := range []int{64, 1024, 65536} {
		chunks := randomChunks(rng, n)
		digests := make([][32]byte, n/2)
		b.Run("generic/chunks="+strconv.Itoa(n), func(b *testing.B) {
			for b.Loop() {
				hashPairsGeneric(digests, chunks)
			}
		})
		b.Run("gohashtree/chunks="+strconv.Itoa(n), func(b *testing.B) {
			for b.Loop() {
				hashPairs(digests, chunks)
			}
		})
	}
}
