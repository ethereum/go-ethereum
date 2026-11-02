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

package ssz_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/ssz"
)

// mustPanic asserts that fn panics.
func mustPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: no panic", name)
		}
	}()
	fn()
}

// nimbusLeafGindex is the progressiveIndexForChunk table of
// nim-ssz-serialization tests/test_proofs.nim, measured from the root of the
// progressive data tree, the form the prover reports.
var nimbusLeafGindex = map[uint64]uint64{
	0:  0b1_0,
	1:  0b1_10_00,
	2:  0b1_10_01,
	3:  0b1_10_10,
	4:  0b1_10_11,
	5:  0b1_110_0000,
	20: 0b1_110_1111,
	21: 0b1_1110_000000,
	84: 0b1_1110_111111,
}

// TestGindexArithmetic pins the index helpers to the examples of
// ethereum/ssz-specs src/ssz/gindex.py and tests/test_proofs.py.
func TestGindexArithmetic(t *testing.T) {
	// gindex_concat: "outer 2, inner 24 -> 40, not 48"; an inner root moves nothing.
	for _, c := range []struct{ outer, inner, want uint64 }{
		{2, 24, 40}, {5, 1, 5}, {1, 7, 7}, {3, 2, 6}, {367, 3, 735},
	} {
		if got := ssz.ConcatGindex(c.outer, c.inner); got != c.want {
			t.Errorf("ConcatGindex(%d, %d) = %d, want %d", c.outer, c.inner, got, c.want)
		}
	}
	// gindex_depth, the number of siblings on a branch.
	for _, c := range []struct {
		index uint64
		want  int
	}{{1, 0}, {2, 1}, {3, 1}, {4, 2}, {105, 6}, {735, 9}, {2856, 11}, {2945, 11}} {
		if got := ssz.GindexDepth(c.index); got != c.want {
			t.Errorf("GindexDepth(%d) = %d, want %d", c.index, got, c.want)
		}
	}
	// Bounded chunks: next_pow2(capacity) + chunk; a capacity of 0 or 1 is the root.
	for _, c := range []struct{ limit, index, want uint64 }{
		{0, 0, 1}, {1, 0, 1}, {2, 0, 2}, {2, 1, 3}, {3, 2, 6}, {3, 3, 7}, {5, 0, 8}, {5, 7, 15}, {8, 7, 15}, {24, 20, 52},
	} {
		if got := ssz.ChunkGindex(c.limit, c.index); got != c.want {
			t.Errorf("ChunkGindex(%d, %d) = %d, want %d", c.limit, c.index, got, c.want)
		}
	}
	mustPanic(t, "GindexDepth(0)", func() { ssz.GindexDepth(0) })
	mustPanic(t, "ConcatGindex(0, 1)", func() { ssz.ConcatGindex(0, 1) })
	mustPanic(t, "ConcatGindex(1, 0)", func() { ssz.ConcatGindex(1, 0) })
	mustPanic(t, "ChunkGindex(2, 2)", func() { ssz.ChunkGindex(2, 2) })
	mustPanic(t, "ChunkGindex(3, 4)", func() { ssz.ChunkGindex(3, 4) })
	mustPanic(t, "ChunkGindex(0, 1)", func() { ssz.ChunkGindex(0, 1) })
}

// TestProgressiveChunkGindex pins the progressive navigation two ways: to
// progressive_chunk_gindex of ethereum/ssz-specs, measured from the type root,
// and to the nim-ssz-serialization vectors, measured from the data tree root,
// which differ by exactly the step into the data tree. TestProveProgressive
// checks that the prover reports the Nimbus form.
func TestProgressiveChunkGindex(t *testing.T) {
	// tests/test_proofs.py TestProgressiveSpine and the gindex.py docstring.
	for chunk, want := range map[uint64]uint64{
		0: 4, 1: 40, 2: 41, 3: 42, 4: 43, 5: 352, 20: 367, 21: 2944, 84: 3007,
	} {
		if got := ssz.ProgressiveChunkGindex(chunk); got != want {
			t.Errorf("ProgressiveChunkGindex(%d) = %d, want %d", chunk, got, want)
		}
	}
	for chunk, inner := range nimbusLeafGindex {
		if got := ssz.ConcatGindex(2, inner); got != ssz.ProgressiveChunkGindex(chunk) {
			t.Errorf("chunk %d: concat(2, nimbus %b) = %d, ProgressiveChunkGindex = %d", chunk, inner, got, ssz.ProgressiveChunkGindex(chunk))
		}
	}
	mustPanic(t, "ProgressiveChunkGindex(1<<40)", func() { ssz.ProgressiveChunkGindex(1 << 40) })
}

// TestPublishedLightClientIndices recomputes, from field positions alone, the
// generalized indices the light-client specs publish and beacon/params
// hardcodes. ssz-specs tests/test_proofs.py pins the same numbers.
func TestPublishedLightClientIndices(t *testing.T) {
	// Pre-Gloas, bounded containers: the Altair BeaconState has 24 fields,
	// finalized_checkpoint at 20 with root as its second field, the sync
	// committees at 22 and 23; the Capella BeaconBlockBody has 11 fields with
	// execution_payload at 9, whose block_hash is field 12 of 15 (Capella)
	// or of 17 (Deneb, body of 12 fields).
	state := func(n, i uint64) uint64 { return ssz.ChunkGindex(n, i) }
	// Gloas, progressive containers: BeaconState positions 20, 22, 23; the
	// BeaconBlockBody has signed_execution_payload_bid at 10, a classic
	// two-field container whose message is a progressive container with
	// parent_block_hash at 0.
	field := ssz.ProgressiveChunkGindex
	for _, c := range []struct {
		name string
		got  uint64
		want uint64
	}{
		{"FINALIZED_ROOT_GINDEX", ssz.ConcatGindex(state(24, 20), state(2, 1)), 105},
		{"CURRENT_SYNC_COMMITTEE_GINDEX", state(24, 22), 54},
		{"NEXT_SYNC_COMMITTEE_GINDEX", state(24, 23), 55},
		{"EXECUTION_PAYLOAD_GINDEX", state(11, 9), 25},
		{"EXECUTION_BLOCK_HASH_GINDEX", ssz.ConcatGindex(state(11, 9), state(15, 12)), 412},
		{"EXECUTION_BLOCK_HASH_GINDEX_DENEB", ssz.ConcatGindex(state(12, 9), state(17, 12)), 812},
		{"FINALIZED_ROOT_GINDEX_GLOAS", ssz.ConcatGindex(field(20), state(2, 1)), 735},
		{"CURRENT_SYNC_COMMITTEE_GINDEX_GLOAS", field(22), 2945},
		{"NEXT_SYNC_COMMITTEE_GINDEX_GLOAS", field(23), 2946},
		{"EXECUTION_BLOCK_HASH_GINDEX_GLOAS", ssz.ConcatGindex(ssz.ConcatGindex(field(10), state(2, 0)), field(0)), 2856},
	} {
		if c.got != c.want {
			t.Errorf("%s: computed %d, spec says %d", c.name, c.got, c.want)
		}
	}
	// The branch lengths the Gloas spec derives as floorlog2 of the constants.
	for _, c := range []struct {
		index uint64
		depth int
	}{{735, 9}, {2945, 11}, {2946, 11}, {2856, 11}} {
		if got := ssz.GindexDepth(c.index); got != c.depth {
			t.Errorf("GindexDepth(%d) = %d, want %d", c.index, got, c.depth)
		}
	}
}
