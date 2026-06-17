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

package state

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// newPreimageTransition returns a BALStateTransition with only the preimage
// accumulator initialised, which is all that AddPreimages/Preimages touch.
func newPreimageTransition() *BALStateTransition {
	return &BALStateTransition{preimages: make(map[common.Hash][]byte)}
}

// TestBALStateTransitionPreimagesUnion verifies that AddPreimages folds the
// preimages contributed by the various block executions (the per-transaction
// copies plus the pre/post-tx execution) into a single set returned by
// Preimages.
func TestBALStateTransitionPreimagesUnion(t *testing.T) {
	s := newPreimageTransition()

	// Empty/nil inputs are no-ops.
	s.AddPreimages(nil)
	s.AddPreimages(map[common.Hash][]byte{})
	if got := len(s.Preimages()); got != 0 {
		t.Fatalf("expected empty set, got %d entries", got)
	}

	s.AddPreimages(map[common.Hash][]byte{
		{0x01}: []byte("pre"),
		{0x02}: []byte("tx1"),
		{0x03}: []byte("tx2"),
		{0x04}: []byte("post"),
	})

	want := map[common.Hash][]byte{
		{0x01}: []byte("pre"),
		{0x02}: []byte("tx1"),
		{0x03}: []byte("tx2"),
		{0x04}: []byte("post"),
	}
	got := s.Preimages()
	if len(got) != len(want) {
		t.Fatalf("preimage count: got %d, want %d", len(got), len(want))
	}
	for h, v := range want {
		if !bytes.Equal(got[h], v) {
			t.Errorf("preimage %x: got %q, want %q", h, got[h], v)
		}
	}
}

// TestBALStateTransitionPreimagesFirstWriteWins verifies that a preimage for a
// hash already present is not overwritten across successive AddPreimages calls,
// matching StateDB.AddPreimage.
func TestBALStateTransitionPreimagesFirstWriteWins(t *testing.T) {
	s := newPreimageTransition()
	key := common.Hash{0xaa}

	s.AddPreimages(map[common.Hash][]byte{key: []byte("first")})
	s.AddPreimages(map[common.Hash][]byte{key: []byte("second")})

	if got := s.Preimages()[key]; !bytes.Equal(got, []byte("first")) {
		t.Fatalf("expected first write to win, got %q", got)
	}
}
