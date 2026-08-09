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

package bintrie

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// leavesOf chunkifies code and returns its non-zero chunks the way the
// snapshot artifact carries them.
func leavesOf(code []byte) []IndexedChunk {
	var (
		chunks = ChunkifyCode(code)
		leaves []IndexedChunk
	)
	for i := 0; i < len(chunks)/32; i++ {
		var c [32]byte
		copy(c[:], chunks[32*i:32*(i+1)])
		if c == ([32]byte{}) {
			continue
		}
		leaves = append(leaves, IndexedChunk{Index: uint64(i), Chunk: c})
	}
	return leaves
}

// TestAssembleCodeRoundTrip pins ChunkifyCode ∘ AssembleCode = identity over
// the shapes that matter: push data crossing chunk boundaries, all-zero
// chunks reading as absence, code past 256 chunks, and ragged tails.
func TestAssembleCodeRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(8347))
	codes := [][]byte{
		{0x01},
		bytes.Repeat([]byte{0x5b}, 31),
		bytes.Repeat([]byte{0x5b}, 32),
		append([]byte{0x60, 0x01, 0x00}, make([]byte, 80)...), // zero tail: absent chunks
		make([]byte, 100), // all-zero code: no leaves at all
		append(bytes.Repeat([]byte{0x00}, 62), 0x5b), // leading absent chunks
		bytes.Repeat([]byte{0x7f}, 33*31),            // PUSH32 walls crossing every boundary
		bytes.Repeat([]byte{0x60, 0x01}, 5000),       // past 256 chunks
	}
	for i := 0; i < 20; i++ {
		code := make([]byte, rng.Intn(2000)+1)
		rng.Read(code)
		codes = append(codes, code)
	}
	for i, code := range codes {
		got, err := AssembleCode(crypto.Keccak256Hash(code), uint32(len(code)), leavesOf(code))
		if err != nil {
			t.Fatalf("code %d: %v", i, err)
		}
		if !bytes.Equal(got, code) {
			t.Fatalf("code %d: reassembly diverges", i)
		}
	}
}

// TestAssembleCodeRejects covers every arm of the code limb: the keccak pins
// the bytes, the re-chunking pins offsets, padding and placement.
func TestAssembleCodeRejects(t *testing.T) {
	// A code whose second chunk carries a non-zero push-data offset.
	code := append(bytes.Repeat([]byte{0x00}, 29), 0x7f) // PUSH32 at offset 29
	code = append(code, bytes.Repeat([]byte{0xaa}, 40)...)
	leaves := leavesOf(code)
	var (
		hash = crypto.Keccak256Hash(code)
		size = uint32(len(code))
	)
	if leaves[1].Chunk[0] == 0 {
		t.Fatal("fixture broken: second chunk carries no push-data offset")
	}
	assemble := func(h common.Hash, s uint32, l []IndexedChunk) error {
		_, err := AssembleCode(h, s, l)
		return err
	}
	if err := assemble(hash, size, leaves); err != nil {
		t.Fatalf("clean fixture rejected: %v", err)
	}

	// A flipped push-data offset byte leaves the code bytes - and so the
	// keccak - intact; only the re-chunking comparison can catch it.
	tampered := append([]IndexedChunk{}, leaves...)
	tampered[1].Chunk[0]++
	if assemble(hash, size, tampered) == nil {
		t.Fatal("a flipped push-data offset byte was accepted")
	}
	// Garbage in the last chunk's zero padding: invisible to the keccak,
	// caught by the re-chunking.
	tampered = append([]IndexedChunk{}, leaves...)
	tampered[len(tampered)-1].Chunk[31] = 0xff
	if size%31 == 0 {
		t.Fatal("fixture broken: last chunk has no padding")
	}
	if assemble(hash, size, tampered) == nil {
		t.Fatal("garbage in the final chunk's padding was accepted")
	}
	// Wrong sizes change the bytes under the hash.
	if assemble(hash, size-1, leaves) == nil || assemble(hash, size+1, leaves) == nil {
		t.Fatal("a wrong code size was accepted")
	}
	// Structural rejections.
	if assemble(hash, 0, nil) == nil {
		t.Fatal("zero code size was accepted")
	}
	if assemble(hash, size, append(append([]IndexedChunk{}, leaves...), IndexedChunk{Index: 1 << 20})) == nil {
		t.Fatal("an out-of-range chunk index was accepted")
	}
	if assemble(hash, size, append(append([]IndexedChunk{}, leaves...), leaves[len(leaves)-1])) == nil {
		t.Fatal("a duplicate chunk index was accepted")
	}
	if assemble(hash, size, []IndexedChunk{{Index: 0}, leaves[1]}) == nil {
		t.Fatal("a spurious all-zero chunk was accepted")
	}
	// A missing non-zero chunk reads as zeros and changes the hash.
	if assemble(hash, size, leaves[1:]) == nil {
		t.Fatal("a missing chunk was accepted")
	}
}
