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
	"errors"
	"math/bits"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie"
)

// A stem normally lives in one groupNode. A tree built from a proof cannot
// always hold it that way: a proof may open only the leaves it covers and
// leave the rest behind a sibling hash, which is representable only as the
// branch-and-leaf structure the group hashes as. These tests pin what the
// engine does with such a stem, because two of the three answers used to be
// silent wrong ones.

// expandGroup rebuilds a two-value group as the branch and leaves foldRange
// hashes it as, mirroring node.go's arithmetic. If it drifts, the hash
// equality asserted below fails and says so.
func expandGroup(g *groupNode, pos int) *branchNode {
	stemBits := 8 * len(g.stem)
	b := 8 - bits.Len8(g.subs[0]^g.subs[1])
	prefix := slice(g.stem, pos, stemBits-pos)
	for k := 0; k < b; k++ {
		prefix = prefix.concat(g.subs[0]>>(7-k)&1, bitstr{})
	}
	return &branchNode{
		prefix: prefix,
		left:   &groupNode{stem: g.stem, subs: []byte{g.subs[0]}, vals: [][]byte{g.vals[0]}},
		right:  &groupNode{stem: g.stem, subs: []byte{g.subs[1]}, vals: [][]byte{g.vals[1]}},
	}
}

// The two sub-index pairs below take different routes through the walks, and
// an earlier version of the guard caught only one of them.
//
// {0x05, 0x80} differ in their first bit, so the expanded branch's prefix ends
// exactly at the stem boundary and the walk reaches the "ran out of stem bits"
// arm. {0x05, 0x07} share six leading bits, so the prefix runs past the
// boundary and the walk reaches the divergence arm instead.
var partialCases = []struct {
	name string
	subs []byte
}{
	{"diverge at first sub bit", []byte{0x05, 0x80}},
	{"diverge inside sub bits", []byte{0x05, 0x07}},
}

// partialFixture returns a stem, its whole group, and the same stem expanded.
func partialFixture(t *testing.T, subs []byte) (stem []byte, whole *groupNode, expanded *branchNode) {
	t.Helper()
	stem = append([]byte{AccountZone}, bytes.Repeat([]byte{0x4c}, AccountKeyLength-2)...)
	whole = &groupNode{
		stem: stem,
		subs: subs,
		vals: [][]byte{
			bytes.Repeat([]byte{0xaa}, 32),
			bytes.Repeat([]byte{0xbb}, 32),
		},
	}
	return stem, whole, expandGroup(whole, 0)
}

func partialTrie(t *testing.T, root binaryNode) *BinaryTrie {
	t.Helper()
	reader, err := trie.NewReader(common.Hash{}, common.Hash{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return &BinaryTrie{
		root:   root,
		reader: reader,
		tracer: trie.NewPrevalueTracer(),
		ops:    newOpTracer(),
	}
}

// TestExpandedStemHashesAsGroup is the foundation everything else rests on:
// the expanded form and the group commit to the same hash, so a proof may
// ship either without changing the root.
func TestExpandedStemHashesAsGroup(t *testing.T) {
	for _, tc := range partialCases {
		t.Run(tc.name, func(t *testing.T) {
			_, whole, expanded := partialFixture(t, tc.subs)
			if got, want := expanded.hashAt(0), whole.hashAt(0); got != want {
				t.Fatalf("expanded stem hashes differently from its group:\n got %x\nwant %x", got, want)
			}
		})
	}
}

// TestPartialStemReadsResolve pins that a covered value reads back through an
// expanded stem.
//
// It did not before. getStem walks by stem, and matchKey stops at the stem's
// end, so the branch carrying the stem's remaining bits plus a few sub-index
// bits never matched and the walk reported the value absent. A stateless
// verifier reading a value the proof shipped as absent executes differently
// from a full node while both roots verify, which is a chain split rather
// than a failed read.
func TestPartialStemReadsResolve(t *testing.T) {
	for _, tc := range partialCases {
		stem, whole, expanded := partialFixture(t, tc.subs)
		for _, shape := range []struct {
			name string
			root binaryNode
		}{
			{"whole group", whole},
			{"expanded stem", expanded},
		} {
			t.Run(tc.name+"/"+shape.name, func(t *testing.T) {
				tr := partialTrie(t, shape.root)
				for i, sub := range whole.subs {
					got, err := tr.getValue(append(append([]byte{}, stem...), sub))
					if err != nil {
						t.Fatalf("sub %#x: %v", sub, err)
					}
					if !bytes.Equal(got, whole.vals[i]) {
						t.Fatalf("sub %#x: got %x, want %x", sub, got, whole.vals[i])
					}
				}
				// A sub-index the stem does not hold still reads absent.
				got, err := tr.getValue(append(append([]byte{}, stem...), 0x42))
				if err != nil {
					t.Fatal(err)
				}
				if got != nil {
					t.Fatalf("absent sub-index returned %x", got)
				}
			})
		}
	}
}

// TestPartialStemRefusesGroupOperations pins that anything needing the whole
// stem says so instead of guessing.
//
// Writing used to panic (the divergence split indexed one byte past the stem)
// and a pure deletion used to return the tree unchanged, reporting a wrong
// root as success. Both are now ErrPartialStem.
func TestPartialStemRefusesGroupOperations(t *testing.T) {
	for _, tc := range partialCases {
		t.Run(tc.name, func(t *testing.T) {
			stem, whole, expanded := partialFixture(t, tc.subs)
			value := bytes.Repeat([]byte{0xcc}, 32)

			t.Run("insert", func(t *testing.T) {
				tr := partialTrie(t, expanded)
				err := tr.UpdateStem(stem, []byte{0x40}, [][]byte{value})
				if !errors.Is(err, ErrPartialStem) {
					t.Fatalf("insert into a partial stem: got %v, want ErrPartialStem", err)
				}
			})

			t.Run("delete", func(t *testing.T) {
				tr := partialTrie(t, expanded)
				before := tr.Hash()
				err := tr.UpdateStem(stem, []byte{whole.subs[0]}, [][]byte{nil})
				if !errors.Is(err, ErrPartialStem) {
					t.Fatalf("delete on a partial stem: got %v, want ErrPartialStem", err)
				}
				if after := tr.Hash(); after != before {
					t.Fatalf("refused delete still moved the root: %x -> %x", before, after)
				}
			})

			t.Run("whole-group scan", func(t *testing.T) {
				tr := partialTrie(t, expanded)
				if _, err := tr.getStemGroup(stem); !errors.Is(err, ErrPartialStem) {
					t.Fatalf("group lookup on a partial stem: got %v, want ErrPartialStem", err)
				}
			})
		})
	}
}

// TestWholeStemUnaffected is the control: the guard added for partial stems
// must not change how an ordinary tree behaves. Without it the tests above
// would pass against an engine that had simply stopped working.
func TestWholeStemUnaffected(t *testing.T) {
	stem, whole, _ := partialFixture(t, []byte{0x05, 0x80})
	tr := partialTrie(t, whole)

	if _, err := tr.getStemGroup(stem); err != nil {
		t.Fatalf("group lookup on a whole stem: %v", err)
	}
	if err := tr.UpdateStem(stem, []byte{0x40}, [][]byte{bytes.Repeat([]byte{0xcc}, 32)}); err != nil {
		t.Fatalf("insert into a whole stem: %v", err)
	}
	got, err := tr.getValue(append(append([]byte{}, stem...), 0x40))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, bytes.Repeat([]byte{0xcc}, 32)) {
		t.Fatalf("inserted value reads back as %x", got)
	}
	// A stem that genuinely is not there stays absent rather than becoming an
	// error, which is what distinguishes the guard from a blanket refusal.
	other := append([]byte{AccountZone}, bytes.Repeat([]byte{0x77}, AccountKeyLength-2)...)
	g, err := tr.getStemGroup(other)
	if err != nil {
		t.Fatalf("absent stem reported an error: %v", err)
	}
	if g != nil {
		t.Fatal("absent stem returned a group")
	}
}
