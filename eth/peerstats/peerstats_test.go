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

// newStats returns a Stats with the given peer ids pre-registered, matching
// the production lifecycle where a peer is registered on connect before any
// of its signals arrive.
func newStats(ids ...string) *Stats {
	s := New()
	for _, id := range ids {
		s.NotifyPeerConnect(id)
	}
	return s
}

// TestPeerLifecycle verifies that an entry exists only between
// NotifyPeerConnect and NotifyPeerDrop: connecting creates it, reconnecting
// keeps its stats, dropping removes it, and signals for a peer without an
// entry never create one, so a peer that has disconnected cannot be
// resurrected by a late signal.
func TestPeerLifecycle(t *testing.T) {
	s := New()

	// Signals for a peer that never connected are ignored.
	s.NotifyBlock(map[string]int{"ghost": 3}, map[string]int{"ghost": 5})
	if n := len(s.GetAllPeerStats()); n != 0 {
		t.Fatalf("signals for unregistered peer must not create entries, got %d", n)
	}

	// Connecting creates an entry; reconnecting keeps its stats.
	s.NotifyPeerConnect("peerA")
	if _, ok := s.GetAllPeerStats()["peerA"]; !ok {
		t.Fatal("expected peerA entry after connect")
	}
	s.NotifyBlock(map[string]int{"peerA": 3}, nil)
	before := s.GetAllPeerStats()["peerA"].RecentIncluded
	s.NotifyPeerConnect("peerA")
	if got := s.GetAllPeerStats()["peerA"].RecentIncluded; got != before {
		t.Fatalf("re-connect wiped stats: got RecentIncluded %f, want %f", got, before)
	}

	// Dropping removes the entry, and a late signal does not recreate it.
	s.NotifyPeerDrop("peerA")
	if _, ok := s.GetAllPeerStats()["peerA"]; ok {
		t.Fatal("NotifyPeerDrop should remove the entry")
	}
	s.NotifyBlock(map[string]int{"peerA": 1}, map[string]int{"peerA": 10})
	if _, ok := s.GetAllPeerStats()["peerA"]; ok {
		t.Fatal("late signal after drop must not recreate the entry")
	}
}

// TestNotifyBlockDecaysFinalized verifies that the finalization EMA decays
// for a peer that earned credits in the past but has no new finalization
// activity. The decay is slow (α=0.0001), so assert monotonic decrease,
// not convergence to zero.
func TestNotifyBlockDecaysFinalized(t *testing.T) {
	s := newStats("peerA")
	s.NotifyBlock(nil, map[string]int{"peerA": 5})
	peak := s.GetAllPeerStats()["peerA"].RecentFinalized
	if peak <= 0 {
		t.Fatalf("expected RecentFinalized>0 after credits, got %f", peak)
	}

	// Credit-free blocks — the EMA must decay monotonically.
	for i := 0; i < 50; i++ {
		s.NotifyBlock(nil, nil)
	}
	after := s.GetAllPeerStats()["peerA"].RecentFinalized
	if after >= peak {
		t.Fatalf("expected RecentFinalized to decay, got %f >= peak %f", after, peak)
	}
}

// TestNotifyBlockInclusionEMAUpdate verifies the EMA formula (1-α)·old + α·count,
// including the pure decay of a peer absent from a block.
func TestNotifyBlockInclusionEMAUpdate(t *testing.T) {
	s := newStats("peerA")
	// Three inclusions: EMA = 0.05 * 3 = 0.15
	s.NotifyBlock(map[string]int{"peerA": 3}, nil)
	got := s.GetAllPeerStats()["peerA"].RecentIncluded
	want := 0.15
	if diff := got - want; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("EMA after one sample: got %f, want %f", got, want)
	}
	// Next block with 10 inclusions: EMA = 0.95*0.15 + 0.05*10 = 0.6425
	s.NotifyBlock(map[string]int{"peerA": 10}, nil)
	got = s.GetAllPeerStats()["peerA"].RecentIncluded
	want = 0.6425
	if diff := got - want; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("EMA after two samples: got %f, want %f", got, want)
	}
	// Block without peerA: pure decay, EMA = 0.95*0.6425 = 0.610375
	s.NotifyBlock(nil, nil)
	got = s.GetAllPeerStats()["peerA"].RecentIncluded
	want = 0.610375
	if diff := got - want; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("EMA after a block without peerA: got %f, want %f", got, want)
	}
}
