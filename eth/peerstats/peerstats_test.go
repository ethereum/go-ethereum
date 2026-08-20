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

// TestNotifyPeerConnectCreatesEntry verifies registration creates a zeroed
// entry and is idempotent (re-registering keeps accumulated stats).
func TestNotifyPeerConnectCreatesEntry(t *testing.T) {
	s := New()
	s.NotifyPeerConnect("peerA")
	if _, ok := s.GetAllPeerStats()["peerA"]; !ok {
		t.Fatal("expected peerA entry after connect")
	}
	// Accumulate some state, then re-connect: stats must be preserved.
	s.NotifyBlock(map[string]int{"peerA": 3}, nil)
	before := s.GetAllPeerStats()["peerA"].RecentIncluded
	s.NotifyPeerConnect("peerA")
	if got := s.GetAllPeerStats()["peerA"].RecentIncluded; got != before {
		t.Fatalf("re-connect wiped stats: got RecentIncluded %f, want %f", got, before)
	}
}

// TestNotifyBlockUpdatesRegisteredPeer verifies that inclusions update the
// EMA of a registered peer.
func TestNotifyBlockUpdatesRegisteredPeer(t *testing.T) {
	s := newStats("peerA")
	s.NotifyBlock(map[string]int{"peerA": 3}, nil)

	ps := s.GetAllPeerStats()["peerA"]
	// EMA after first block: (1-0.05)*0 + 0.05*3 = 0.15
	if ps.RecentIncluded <= 0 {
		t.Fatalf("expected RecentIncluded > 0 after inclusion, got %f", ps.RecentIncluded)
	}
}

// TestNotifyBlockIgnoresUnregisteredPeer verifies that inclusions (and
// finalization credits) for a peer with no entry never create one — a tx
// delivered by a peer that has since disconnected cannot resurrect its stats.
func TestNotifyBlockIgnoresUnregisteredPeer(t *testing.T) {
	s := New()
	s.NotifyBlock(map[string]int{"ghost": 3}, map[string]int{"ghost": 5})
	if n := len(s.GetAllPeerStats()); n != 0 {
		t.Fatalf("signals for unregistered peer must not create entries, got %d", n)
	}
}

// TestNotifyBlockDecaysKnownPeers verifies that registered peers get their
// RecentIncluded EMA decayed when they have no inclusions in a block.
func TestNotifyBlockDecaysKnownPeers(t *testing.T) {
	s := newStats("peerA")
	s.NotifyBlock(map[string]int{"peerA": 3}, nil)
	initial := s.GetAllPeerStats()["peerA"].RecentIncluded

	// Empty block — peerA should decay.
	s.NotifyBlock(nil, nil)
	after := s.GetAllPeerStats()["peerA"].RecentIncluded

	if after >= initial {
		t.Fatalf("expected decay, got %f >= %f", after, initial)
	}
}

// TestNotifyBlockDropThenFinalizeNoResurrect verifies the full drop→finalize
// sequence: a dropped peer doesn't come back via finalization credits.
func TestNotifyBlockDropThenFinalizeNoResurrect(t *testing.T) {
	s := newStats("peerA")
	s.NotifyBlock(map[string]int{"peerA": 1}, nil)
	s.NotifyPeerDrop("peerA")
	s.NotifyBlock(nil, map[string]int{"peerA": 10})

	if stats := s.GetAllPeerStats(); len(stats) != 0 {
		t.Fatalf("dropped peer must not be resurrected, got %d peers", len(stats))
	}
}

// TestNotifyBlockFinalizationCredits an existing peer.
func TestNotifyBlockFinalizationCredits(t *testing.T) {
	s := newStats("peerA")
	s.NotifyBlock(map[string]int{"peerA": 1}, nil)
	s.NotifyBlock(nil, map[string]int{"peerA": 3})

	// RecentFinalized is a slow EMA, not a cumulative count: assert it
	// moved in the positive direction, not the exact value.
	if got := s.GetAllPeerStats()["peerA"].RecentFinalized; got <= 0 {
		t.Fatalf("expected RecentFinalized>0 after credits, got %f", got)
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

// TestNotifyBlockInclusionEMAUpdate verifies the EMA formula (1-α)·old + α·count.
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
}

// TestNotifyPeerDropClearsStats verifies disconnect cleanup removes the
// peer's entry.
func TestNotifyPeerDropClearsStats(t *testing.T) {
	s := newStats("peerA")
	s.NotifyBlock(map[string]int{"peerA": 1}, nil)
	s.NotifyPeerDrop("peerA")

	if _, ok := s.GetAllPeerStats()["peerA"]; ok {
		t.Fatal("NotifyPeerDrop should remove the entry")
	}
}
