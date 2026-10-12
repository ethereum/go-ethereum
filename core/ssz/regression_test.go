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

// Regression tests named after bugs fixed in status-im/nim-ssz-serialization,
// the reference implementation of the progressive types. Nimbus hit them in
// its cached hash trees and its hashing fast path, neither of which this
// package has, so each test pins the input shape that exposed the bug rather
// than the bug itself.

import (
	"math/rand"
	"testing"
)

// TestRegressionNonAlignedInput: nim-ssz-serialization PR 155 fixed a digest
// fast path that mishandled inputs other than 64 bytes long. Byte lists of
// every length modulo a chunk must root the same as the recursion over their
// padded chunks.
func TestRegressionNonAlignedInput(t *testing.T) {
	rng := rand.New(rand.NewSource(155))
	for n := 0; n < 200; n++ {
		data := make([]byte, n)
		rng.Read(data)
		want := MixInLength(naiveMerkleizeProgressive(Pack(data), 1), uint64(n))
		if got := HashTreeRootProgressiveBytes(data); got != want {
			t.Fatalf("%d bytes: root %x, want %x", n, got, want)
		}
	}
}

// TestRegressionCompositePadding: PR 162 fixed composite lists whose cached
// tree dropped the zero padding of the last, partially filled subtree. One
// element past each boundary leaves most of the new subtree empty.
func TestRegressionCompositePadding(t *testing.T) {
	rng := rand.New(rand.NewSource(162))
	for _, n := range []int{2, 3, 6, 22, 86, 342} {
		roots := randomChunks(rng, n)
		want := MixInLength(naiveMerkleizeProgressive(roots, 1), uint64(n))
		if got := HashTreeRootProgressiveRoots(roots); got != want {
			t.Fatalf("%d composites: root %x, want %x", n, got, want)
		}
	}
}

// TestRegressionSingleChunk: PR 163 fixed a depth computation that
// overflowed on a single-chunk tree. The first subtree is the chunk itself,
// hashed once against the zero terminator.
func TestRegressionSingleChunk(t *testing.T) {
	rng := rand.New(rand.NewSource(163))
	chunk := randomChunks(rng, 1)[0]
	if sub := Merkleize([][32]byte{chunk}, 1); sub != chunk {
		t.Fatalf("single-chunk subtree root %x, want the chunk itself", sub)
	}
	if got, want := MerkleizeProgressive([][32]byte{chunk}), hashPair(chunk, [32]byte{}); got != want {
		t.Fatalf("single-chunk root %x, want %x", got, want)
	}
}

// TestRegressionSubtreeBoundaries: PRs 146 and 150 fixed cache invalidation
// when a list grew or shrank across a subtree boundary, exercised at the
// lengths 21844, 340, 84, 20 and 1. Roots at those lengths and their
// neighbours must match the recursion and must all differ from one another.
func TestRegressionSubtreeBoundaries(t *testing.T) {
	rng := rand.New(rand.NewSource(150))
	chunks := randomChunks(rng, 21846)
	seen := make(map[[32]byte]int)
	for _, n := range []int{0, 1, 2, 20, 21, 22, 84, 85, 86, 340, 341, 342, 21844, 21845, 21846} {
		got := MerkleizeProgressive(chunks[:n])
		if want := naiveMerkleizeProgressive(chunks[:n], 1); got != want {
			t.Fatalf("n=%d: root %x, want %x", n, got, want)
		}
		if prev, dup := seen[got]; dup {
			t.Fatalf("lengths %d and %d share a root", prev, n)
		}
		seen[got] = n
	}
}
