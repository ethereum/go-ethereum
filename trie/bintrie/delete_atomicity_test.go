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
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// Emptying a group hands its parent a collapse, which resolves the surviving
// sibling and can fail - routinely, on a witness-backed replay reading lazily.
// The write therefore decides whether it empties the group before mutating.

// atomicityTree returns a branch whose 0 side is a two-value group and whose 1
// side cannot be resolved, plus that group and its stem.
func atomicityTree(t *testing.T) (*BinaryTrie, *groupNode, []byte) {
	t.Helper()
	stem := append([]byte{AccountZone}, bytes.Repeat([]byte{0x00}, AccountKeyLength-2)...)
	group := &groupNode{
		stem: stem,
		subs: []byte{0, 1},
		vals: [][]byte{bytes.Repeat([]byte{0xaa}, 32), bytes.Repeat([]byte{0xbb}, 32)},
	}
	root := &branchNode{
		prefix: slice(stem, 0, 8),
		left:   group,
		right:  hashedNode{0xde, 0xad},
	}
	return partialTrie(t, root), group, stem
}

// TestRefusedDeleteChangesNothing pins that a delete which cannot complete
// leaves the tree as it found it: the group used to be emptied in place, and
// the next write to the stem rooted state that had silently gone.
func TestRefusedDeleteChangesNothing(t *testing.T) {
	tr, group, stem := atomicityTree(t)
	extra := bytes.Repeat([]byte{0xcc}, 32)

	if err := tr.UpdateStem(stem, []byte{0, 1}, [][]byte{nil, nil}); err == nil {
		t.Fatal("a delete whose collapse cannot resolve its sibling succeeded")
	}
	if len(group.subs) != 2 {
		t.Fatalf("the refused delete left the group holding %d values, want 2", len(group.subs))
	}
	// IntermediateRoot latches a failed write and carries on through the rest of
	// the block's mutations, so the next write is what surfaces the corruption.
	if err := tr.UpdateStem(stem, []byte{2}, [][]byte{extra}); err != nil {
		t.Fatalf("the follow-up write failed: %v", err)
	}
	ref, _, _ := atomicityTree(t)
	if err := ref.UpdateStem(stem, []byte{2}, [][]byte{extra}); err != nil {
		t.Fatal(err)
	}
	if got, want := tr.Hash(), ref.Hash(); got != want {
		t.Fatalf("the refused delete was half applied:\n got %x\nwant %x", got, want)
	}
}

// TestRefusedDeleteSurvivesHashing pins the same refusal against a fold: the
// panic this used to raise landed on a worker goroutine no caller can recover,
// so an incomplete witness took the process down instead of being rejected.
func TestRefusedDeleteSurvivesHashing(t *testing.T) {
	tr, _, stem := atomicityTree(t)

	if err := tr.UpdateStem(stem, []byte{0, 1}, [][]byte{nil, nil}); err == nil {
		t.Fatal("a delete whose collapse cannot resolve its sibling succeeded")
	}
	// A neighbouring stem re-parents the group under a fresh dirty branch, which
	// is what makes the fold descend into it again.
	neighbour := append([]byte{AccountZone}, bytes.Repeat([]byte{0x00}, AccountKeyLength-2)...)
	neighbour[5] = 0x33
	if err := tr.UpdateStem(neighbour, []byte{0}, [][]byte{bytes.Repeat([]byte{0xdd}, 32)}); err != nil {
		t.Fatalf("the neighbouring write failed: %v", err)
	}
	tr.Hash()
}

// TestEmptyGroupFoldsAsEmpty pins the fold's own guard, so the crash cannot
// come back through a path this file does not model.
func TestEmptyGroupFoldsAsEmpty(t *testing.T) {
	g := &groupNode{stem: append([]byte{AccountZone}, bytes.Repeat([]byte{0x00}, AccountKeyLength-2)...)}
	if got := g.hashAt(0); got != (common.Hash{}) {
		t.Fatalf("an empty leaf set hashed to %x, want the empty subtree", got)
	}
}

// TestEmptiedBy pins which batches empty a group, since deciding wrongly either
// loses the collapse or applies a write the tree then has to unpick.
func TestEmptiedBy(t *testing.T) {
	val := bytes.Repeat([]byte{0xaa}, 32)
	g := &groupNode{
		stem: append([]byte{AccountZone}, bytes.Repeat([]byte{0x00}, AccountKeyLength-2)...),
		subs: []byte{4, 9},
		vals: [][]byte{val, val},
	}
	for _, tc := range []struct {
		name string
		subs []byte
		vals [][]byte
		want bool
	}{
		{"every value deleted", []byte{4, 9}, [][]byte{nil, nil}, true},
		{"deleted plus an absent sub", []byte{4, 9, 7}, [][]byte{nil, nil, nil}, true},
		{"one value left", []byte{4}, [][]byte{nil}, false},
		{"one overwritten rather than deleted", []byte{4, 9}, [][]byte{nil, val}, false},
		{"an insertion alongside the deletes", []byte{4, 9, 7}, [][]byte{nil, nil, val}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := emptiedBy(g, tc.subs, tc.vals); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
