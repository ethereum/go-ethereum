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

package peerstats

import (
	"testing"
)

// TestNotifyBlockCreatesEntryOnInclusion verifies that a positive
// inclusion delta creates the peer's entry — the path by which
// peerstats learns about a peer — and feeds its inclusion EMA.
func TestNotifyBlockCreatesEntryOnInclusion(t *testing.T) {
	s := New()
	s.NotifyBlock(map[string]int{"peerA": 2}, nil)

	ps, ok := s.GetAllPeerStats()["peerA"]
	if !ok {
		t.Fatal("inclusion delta should create the peer entry")
	}
	if ps.RecentIncluded == 0 {
		t.Error("RecentIncluded should be non-zero after an inclusion")
	}
}

// TestNotifyBlockInclusionEMAUpdate verifies the EMA formula for a
// single step from a zeroed entry.
func TestNotifyBlockInclusionEMAUpdate(t *testing.T) {
	s := New()
	s.NotifyBlock(map[string]int{"peerA": 10}, nil)

	got := s.GetAllPeerStats()["peerA"].RecentIncluded
	want := emaAlpha * 10.0
	if diff := got - want; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("RecentIncluded after one block: got %f, want %f", got, want)
	}
}

// TestNotifyBlockDecaysKnownPeers verifies that a block without a
// tracked peer's inclusions decays its EMA (zero contribution).
func TestNotifyBlockDecaysKnownPeers(t *testing.T) {
	s := New()
	s.NotifyBlock(map[string]int{"peerA": 10}, nil)
	first := s.GetAllPeerStats()["peerA"].RecentIncluded

	s.NotifyBlock(nil, nil)
	second := s.GetAllPeerStats()["peerA"].RecentIncluded
	if second >= first {
		t.Fatalf("RecentIncluded should decay on an empty block: %f >= %f", second, first)
	}
}

// TestNotifyBlockFinalizationCredits verifies finalization deltas feed
// the slow EMA of a tracked peer.
func TestNotifyBlockFinalizationCredits(t *testing.T) {
	s := New()
	s.NotifyBlock(map[string]int{"peerA": 1}, nil) // create via inclusion
	s.NotifyBlock(nil, map[string]int{"peerA": 5})

	if got := s.GetAllPeerStats()["peerA"].RecentFinalized; got == 0 {
		t.Fatal("RecentFinalized should be non-zero after finalization credits")
	}
}

// TestNotifyBlockDecaysFinalized verifies the finalization EMA decays
// monotonically across credit-free blocks.
func TestNotifyBlockDecaysFinalized(t *testing.T) {
	s := New()
	s.NotifyBlock(map[string]int{"peerA": 1}, map[string]int{"peerA": 5})
	prev := s.GetAllPeerStats()["peerA"].RecentFinalized
	if prev == 0 {
		t.Fatal("seeding RecentFinalized failed")
	}
	for i := 0; i < 50; i++ {
		s.NotifyBlock(nil, nil)
		cur := s.GetAllPeerStats()["peerA"].RecentFinalized
		if cur >= prev {
			t.Fatalf("RecentFinalized should decay monotonically: block %d: %f >= %f", i, cur, prev)
		}
		prev = cur
	}
}

// TestNotifyBlockDropThenFinalizeNoResurrect verifies that finalization
// credit arriving after NotifyPeerDrop does not resurrect the entry —
// historical data must not repopulate the map.
func TestNotifyBlockDropThenFinalizeNoResurrect(t *testing.T) {
	s := New()
	s.NotifyBlock(map[string]int{"peerA": 1}, nil)
	s.NotifyPeerDrop("peerA")
	s.NotifyBlock(nil, map[string]int{"peerA": 3})

	if _, ok := s.GetAllPeerStats()["peerA"]; ok {
		t.Fatal("finalization credit must not resurrect a dropped peer")
	}
}

// TestNotifyPeerDropClearsStats verifies disconnect cleanup removes the
// peer's entry.
func TestNotifyPeerDropClearsStats(t *testing.T) {
	s := New()
	s.NotifyBlock(map[string]int{"peerA": 1}, nil)
	s.NotifyPeerDrop("peerA")

	if _, ok := s.GetAllPeerStats()["peerA"]; ok {
		t.Fatal("NotifyPeerDrop should remove the entry")
	}
}
