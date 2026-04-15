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
	"time"
)

// TestNotifyBlockBootstrapsFromInclusions verifies that a peer with a positive
// inclusion count in the first NotifyBlock gets a stats entry created.
func TestNotifyBlockBootstrapsFromInclusions(t *testing.T) {
	s := New()
	s.NotifyBlock(map[string]int{"peerA": 3}, nil)

	stats := s.GetAllPeerStats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 peer entry, got %d", len(stats))
	}
	ps, ok := stats["peerA"]
	if !ok {
		t.Fatal("expected peerA entry")
	}
	// EMA after first block: (1-0.05)*0 + 0.05*3 = 0.15
	if ps.RecentIncluded <= 0 {
		t.Fatalf("expected RecentIncluded > 0 after inclusion, got %f", ps.RecentIncluded)
	}
}

// TestNotifyBlockDecaysKnownPeers verifies that peers already tracked get their
// RecentIncluded EMA decayed when they have no inclusions in a block.
func TestNotifyBlockDecaysKnownPeers(t *testing.T) {
	s := New()
	// Seed peerA with an inclusion.
	s.NotifyBlock(map[string]int{"peerA": 3}, nil)
	initial := s.GetAllPeerStats()["peerA"].RecentIncluded

	// Empty block — peerA should decay.
	s.NotifyBlock(nil, nil)
	after := s.GetAllPeerStats()["peerA"].RecentIncluded

	if after >= initial {
		t.Fatalf("expected decay, got %f >= %f", after, initial)
	}
}

// TestNotifyBlockDoesNotResurrectDroppedPeers verifies that finalization
// credits to a peer with no entry don't create one.
func TestNotifyBlockDoesNotResurrectFromFinalization(t *testing.T) {
	s := New()
	s.NotifyBlock(nil, map[string]int{"peerA": 5})

	if stats := s.GetAllPeerStats(); len(stats) != 0 {
		t.Fatalf("finalization credits must not create entries, got %d peers", len(stats))
	}
}

// TestNotifyBlockDropThenFinalizeNoResurrect verifies the full drop→finalize
// sequence: a dropped peer doesn't come back via finalization credits.
func TestNotifyBlockDropThenFinalizeNoResurrect(t *testing.T) {
	s := New()
	s.NotifyBlock(map[string]int{"peerA": 1}, nil)
	s.NotifyPeerDrop("peerA")
	s.NotifyBlock(nil, map[string]int{"peerA": 10})

	if stats := s.GetAllPeerStats(); len(stats) != 0 {
		t.Fatalf("dropped peer must not be resurrected, got %d peers", len(stats))
	}
}

// TestNotifyBlockFinalizationCredits an existing peer.
func TestNotifyBlockFinalizationCredits(t *testing.T) {
	s := New()
	s.NotifyBlock(map[string]int{"peerA": 1}, nil)
	s.NotifyBlock(nil, map[string]int{"peerA": 3})

	// RecentFinalized is a slow EMA, not a cumulative count: assert it
	// moved in the positive direction, not the exact value.
	if got := s.GetAllPeerStats()["peerA"].RecentFinalized; got <= 0 {
		t.Fatalf("expected RecentFinalized>0 after credits, got %f", got)
	}
}

// TestNotifyBlockInclusionEMAUpdate verifies the EMA formula (1-α)·old + α·count.
func TestNotifyBlockInclusionEMAUpdate(t *testing.T) {
	s := New()
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

// TestNotifyRequestLatencyFirstSampleBootstrap asserts that the first
// latency sample seeds the EMA directly.
func TestNotifyRequestLatencyFirstSampleBootstrap(t *testing.T) {
	s := New()
	s.NotifyRequestLatency("peerA", 200*time.Millisecond)

	ps := s.GetAllPeerStats()["peerA"]
	if ps.RequestLatencyEMA != 200*time.Millisecond {
		t.Fatalf("expected first sample to seed EMA at 200ms, got %v", ps.RequestLatencyEMA)
	}
	if ps.RequestSamples != 1 {
		t.Fatalf("expected RequestSamples=1, got %d", ps.RequestSamples)
	}
}

// TestNotifyRequestLatencyEMAUpdate verifies the EMA formula for latency.
func TestNotifyRequestLatencyEMAUpdate(t *testing.T) {
	s := New()
	s.NotifyRequestLatency("peerA", 100*time.Millisecond)
	s.NotifyRequestLatency("peerA", 1000*time.Millisecond)

	// Expected: 0.99*100ms + 0.01*1000ms = 109ms
	got := s.GetAllPeerStats()["peerA"].RequestLatencyEMA
	want := 109 * time.Millisecond
	delta := got - want
	if delta < 0 {
		delta = -delta
	}
	if delta > 1*time.Microsecond {
		t.Fatalf("EMA mismatch: got %v, want %v", got, want)
	}
	if samples := s.GetAllPeerStats()["peerA"].RequestSamples; samples != 2 {
		t.Fatalf("expected RequestSamples=2, got %d", samples)
	}
}

// TestNotifyRequestLatencySlowConvergence verifies the slow alpha
// damps convergence under sustained timeouts.
func TestNotifyRequestLatencySlowConvergence(t *testing.T) {
	s := New()
	s.NotifyRequestLatency("peerA", 100*time.Millisecond)
	for i := 0; i < 50; i++ {
		s.NotifyRequestLatency("peerA", 5*time.Second)
	}
	got := s.GetAllPeerStats()["peerA"].RequestLatencyEMA
	if got < 1*time.Second {
		t.Fatalf("EMA did not move enough under sustained timeouts, got %v", got)
	}
	if got > 3*time.Second {
		t.Fatalf("EMA converged too fast for slow alpha=0.01, got %v", got)
	}
}

// TestNotifyPeerDropClearsStats verifies that a dropped peer disappears
// from GetAllPeerStats.
func TestNotifyPeerDropClearsStats(t *testing.T) {
	s := New()
	s.NotifyRequestLatency("peerA", 200*time.Millisecond)
	s.NotifyPeerDrop("peerA")

	if _, ok := s.GetAllPeerStats()["peerA"]; ok {
		t.Fatal("peerA stats should be removed after NotifyPeerDrop")
	}
}

// TestStaleRequestLatencyAfterDrop documents the accepted behavior: a
// late sample after NotifyPeerDrop recreates a 1-sample entry. The
// dropper's MinLatencySamples=10 guard ensures this is harmless.
func TestStaleRequestLatencyAfterDrop(t *testing.T) {
	s := New()
	s.NotifyRequestLatency("peerA", 200*time.Millisecond)
	s.NotifyPeerDrop("peerA")
	// Late sample racing with the drop.
	s.NotifyRequestLatency("peerA", 50*time.Millisecond)

	ps := s.GetAllPeerStats()["peerA"]
	if ps.RequestSamples != 1 {
		t.Fatalf("expected fresh RequestSamples=1, got %d", ps.RequestSamples)
	}
	if ps.RequestLatencyEMA != 50*time.Millisecond {
		t.Fatalf("expected fresh bootstrap at 50ms, got %v", ps.RequestLatencyEMA)
	}
	// The dropper's MinLatencySamples guard (in eth/dropper.go) prevents
	// this 1-sample entry from earning latency-based protection.
}

// TestMultiplePeersIsolated verifies per-peer isolation across signal types.
func TestMultiplePeersIsolated(t *testing.T) {
	s := New()
	s.NotifyBlock(map[string]int{"peerA": 5, "peerB": 0}, nil)
	s.NotifyRequestLatency("peerA", 100*time.Millisecond)
	s.NotifyRequestLatency("peerB", 5*time.Second)
	s.NotifyBlock(nil, map[string]int{"peerA": 2})

	stats := s.GetAllPeerStats()
	// Only peerA receives finalization credits; peerB's EMA stays at zero
	// (no credits, pure decay from zero).
	if stats["peerA"].RecentFinalized <= 0 || stats["peerB"].RecentFinalized != 0 {
		t.Errorf("finalization leaked: A=%f B=%f", stats["peerA"].RecentFinalized, stats["peerB"].RecentFinalized)
	}
	if stats["peerA"].RequestLatencyEMA != 100*time.Millisecond {
		t.Errorf("peerA latency: got %v, want 100ms", stats["peerA"].RequestLatencyEMA)
	}
	if stats["peerB"].RequestLatencyEMA != 5*time.Second {
		t.Errorf("peerB latency: got %v, want 5s", stats["peerB"].RequestLatencyEMA)
	}
}
