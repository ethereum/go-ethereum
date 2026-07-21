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

// TestNotifyRequestResultFirstSampleBootstrap asserts that the first
// latency sample seeds the EMA directly.
func TestNotifyRequestResultFirstSampleBootstrap(t *testing.T) {
	s := New()
	s.NotifyRequestResult("peerA", 200*time.Millisecond, false)

	ps := s.GetAllPeerStats()["peerA"]
	if ps.RequestLatencyEMA != 200*time.Millisecond {
		t.Fatalf("expected first sample to seed EMA at 200ms, got %v", ps.RequestLatencyEMA)
	}
	if ps.RequestSuccesses != 1 {
		t.Fatalf("expected RequestSuccesses=1, got %d", ps.RequestSuccesses)
	}
	if ps.RequestTimeouts != 0 {
		t.Fatalf("expected RequestTimeouts=0, got %d", ps.RequestTimeouts)
	}
}

// TestNotifyRequestResultEMAUpdate verifies the EMA formula for latency.
func TestNotifyRequestResultEMAUpdate(t *testing.T) {
	s := New()
	s.NotifyRequestResult("peerA", 100*time.Millisecond, false)
	s.NotifyRequestResult("peerA", 1000*time.Millisecond, false)

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
	ps := s.GetAllPeerStats()["peerA"]
	if ps.RequestSuccesses != 2 {
		t.Fatalf("expected RequestSuccesses=2, got %d", ps.RequestSuccesses)
	}
}

// TestNotifyRequestResultSlowConvergence verifies the slow alpha
// damps convergence under sustained timeouts.
func TestNotifyRequestResultSlowConvergence(t *testing.T) {
	s := New()
	s.NotifyRequestResult("peerA", 100*time.Millisecond, false)
	for i := 0; i < 50; i++ {
		s.NotifyRequestResult("peerA", 5*time.Second, false)
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
	s.NotifyRequestResult("peerA", 200*time.Millisecond, false)
	s.NotifyPeerDrop("peerA")

	if _, ok := s.GetAllPeerStats()["peerA"]; ok {
		t.Fatal("peerA stats should be removed after NotifyPeerDrop")
	}
}

// TestPruneRemovesDisconnectedPeers verifies Prune drops entries for peers
// absent from the keep set (e.g. an orphan recreated by a signal that raced
// with NotifyPeerDrop) while retaining still-connected peers.
func TestPruneRemovesDisconnectedPeers(t *testing.T) {
	s := New()
	s.NotifyRequestResult("connected", 100*time.Millisecond, false)
	s.NotifyRequestResult("orphan", 100*time.Millisecond, false)

	s.Prune(map[string]bool{"connected": true})

	stats := s.GetAllPeerStats()
	if _, ok := stats["orphan"]; ok {
		t.Fatal("orphan stats should be pruned when not in keep set")
	}
	if _, ok := stats["connected"]; !ok {
		t.Fatal("connected peer stats should survive pruning")
	}
}

// TestPruneEmptyKeepClearsAll verifies an empty keep set removes every entry.
func TestPruneEmptyKeepClearsAll(t *testing.T) {
	s := New()
	s.NotifyRequestResult("peerA", 100*time.Millisecond, false)
	s.NotifyRequestResult("peerB", 100*time.Millisecond, false)

	s.Prune(map[string]bool{})

	if n := len(s.GetAllPeerStats()); n != 0 {
		t.Fatalf("expected all entries pruned, got %d", n)
	}
}

// TestStaleRequestLatencyAfterDrop documents the accepted behavior: a
// late sample after NotifyPeerDrop recreates a 1-sample entry. The
// dropper's MinLatencyActivity guard ensures this is harmless, and the
// dropper's periodic Prune reclaims the orphan.
func TestStaleRequestLatencyAfterDrop(t *testing.T) {
	s := New()
	s.NotifyRequestResult("peerA", 200*time.Millisecond, false)
	s.NotifyPeerDrop("peerA")
	// Late sample racing with the drop.
	s.NotifyRequestResult("peerA", 50*time.Millisecond, false)

	ps := s.GetAllPeerStats()["peerA"]
	if ps.RequestSuccesses != 1 {
		t.Fatalf("expected fresh RequestSuccesses=1, got %d", ps.RequestSuccesses)
	}
	if ps.RequestLatencyEMA != 50*time.Millisecond {
		t.Fatalf("expected fresh bootstrap at 50ms, got %v", ps.RequestLatencyEMA)
	}
	// The dropper's MinLatencyActivity guard (in eth/dropper.go) prevents
	// this 1-sample entry from earning latency-based protection.
}

// TestMultiplePeersIsolated verifies per-peer isolation across signal types.
func TestMultiplePeersIsolated(t *testing.T) {
	s := New()
	s.NotifyBlock(map[string]int{"peerA": 5, "peerB": 0}, nil)
	s.NotifyRequestResult("peerA", 100*time.Millisecond, false)
	s.NotifyRequestResult("peerB", 5*time.Second, false)
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

// TestLatencyActivityAccumulatesAndDecays verifies that a block with an
// accepted delivery folds a positive presence bit into the activity EMA, and
// that a subsequent delivery-free block decays it (not resets it).
func TestLatencyActivityAccumulatesAndDecays(t *testing.T) {
	s := New()
	for i := 0; i < 10; i++ {
		s.NotifyRequestResult("peerA", 100*time.Millisecond, false)
	}
	s.NotifyBlock(nil, nil)

	folded := s.GetAllPeerStats()["peerA"].LatencyActivity
	if folded <= 0 {
		t.Fatalf("expected positive activity after folding samples, got %f", folded)
	}

	// An empty block: nothing delivered, pure decay.
	s.NotifyBlock(nil, nil)
	decayed := s.GetAllPeerStats()["peerA"].LatencyActivity
	if decayed >= folded {
		t.Fatalf("expected activity to decay on empty block, got %f >= %f", decayed, folded)
	}
	if decayed <= 0 {
		t.Fatalf("expected gradual decay, not reset, got %f", decayed)
	}
}

// TestLatencyActivityCapsBurst verifies the per-block contribution is capped:
// a block with a burst of many accepted deliveries produces the same activity
// as a block with a single delivery, so eligibility cannot be front-loaded.
func TestLatencyActivityCapsBurst(t *testing.T) {
	single := New()
	single.NotifyRequestResult("peerA", 50*time.Millisecond, false)
	single.NotifyBlock(nil, nil)

	burst := New()
	for i := 0; i < 100; i++ {
		burst.NotifyRequestResult("peerA", 50*time.Millisecond, false)
	}
	burst.NotifyBlock(nil, nil)

	got := burst.GetAllPeerStats()["peerA"].LatencyActivity
	want := single.GetAllPeerStats()["peerA"].LatencyActivity
	if got != want {
		t.Fatalf("burst of 100 deliveries in one block should equal a single delivery: got %f, want %f", got, want)
	}
	// And one block's contribution must be well under the eligibility gate,
	// so a single burst block cannot by itself confer protection.
	if got >= MinLatencyActivity {
		t.Fatalf("one block should not reach the eligibility gate, got %f >= %f", got, MinLatencyActivity)
	}
}

// TestLatencyActivityGateReachable verifies that a peer sustaining one
// accepted delivery per block crosses MinLatencyActivity within a
// reasonable number of blocks (steady state for 1/block is 1.0).
func TestLatencyActivityGateReachable(t *testing.T) {
	s := New()
	for i := 0; i < 20; i++ {
		s.NotifyRequestResult("peerA", 100*time.Millisecond, false)
		s.NotifyBlock(nil, nil)
	}
	if got := s.GetAllPeerStats()["peerA"].LatencyActivity; got < MinLatencyActivity {
		t.Fatalf("sustained 1 sample/block should reach eligibility, got %f < %f", got, MinLatencyActivity)
	}
}

// TestTimeoutDoesNotFeedActivity verifies that timeouts update the EMA and
// counters but never contribute to the activity rate — a peer cannot become
// protection-eligible by timing out.
func TestTimeoutDoesNotFeedActivity(t *testing.T) {
	s := New()
	for i := 0; i < 50; i++ {
		s.NotifyRequestResult("peerA", 5*time.Second, true)
		s.NotifyBlock(nil, nil)
	}
	ps := s.GetAllPeerStats()["peerA"]
	if ps.LatencyActivity != 0 {
		t.Fatalf("timeouts must not feed activity, got %f", ps.LatencyActivity)
	}
	if ps.RequestTimeouts == 0 {
		t.Fatal("expected timeout counter to advance")
	}
}

// TestLatencyStateForgottenAfterSilence verifies that once a silent peer's
// activity fully decays, its latency state (EMA and counters) is reset —
// a frozen fast EMA from a past active period cannot be re-armed later by
// rebuilding activity alone.
func TestLatencyStateForgottenAfterSilence(t *testing.T) {
	s := New()
	s.NotifyRequestResult("peerA", 50*time.Millisecond, false)
	s.NotifyBlock(nil, nil)
	if s.GetAllPeerStats()["peerA"].RequestLatencyEMA != 50*time.Millisecond {
		t.Fatal("expected EMA seeded before silence")
	}

	// Enough empty blocks for the activity to decay below the reset
	// threshold (~200 blocks from a single sample at alpha=0.014).
	for i := 0; i < 400; i++ {
		s.NotifyBlock(nil, nil)
	}

	ps := s.GetAllPeerStats()["peerA"]
	if ps.RequestLatencyEMA != 0 || ps.RequestSuccesses != 0 || ps.RequestTimeouts != 0 || ps.LatencyActivity != 0 {
		t.Fatalf("expected latency state forgotten after long silence, got %+v", ps)
	}

	// A returning peer starts over: the next sample re-seeds the EMA.
	s.NotifyRequestResult("peerA", 300*time.Millisecond, false)
	if got := s.GetAllPeerStats()["peerA"].RequestLatencyEMA; got != 300*time.Millisecond {
		t.Fatalf("expected fresh bootstrap after reset, got %v", got)
	}
}

// TestSilenceResetKeepsTimeoutRecord verifies that the silence reset forgets
// the fast EMA and success count but PRESERVES the accumulated timeout penalty
// counter — a peer with a success history cannot launder away its timeout
// record merely by going silent long enough to decay.
func TestSilenceResetKeepsTimeoutRecord(t *testing.T) {
	s := New()
	// Build up activity through sustained successes, then record timeouts.
	for i := 0; i < 20; i++ {
		s.NotifyRequestResult("peerA", 50*time.Millisecond, false)
		s.NotifyBlock(nil, nil)
	}
	s.NotifyRequestResult("peerA", 5*time.Second, true)
	s.NotifyRequestResult("peerA", 5*time.Second, true)
	if got := s.GetAllPeerStats()["peerA"].RequestTimeouts; got != 2 {
		t.Fatalf("setup: expected 2 timeouts, got %d", got)
	}

	// Go silent long enough for activity to decay below the reset threshold.
	for i := 0; i < 600; i++ {
		s.NotifyBlock(nil, nil)
	}

	ps := s.GetAllPeerStats()["peerA"]
	if ps.RequestLatencyEMA != 0 {
		t.Fatalf("fast EMA should be forgotten on reset, got %v", ps.RequestLatencyEMA)
	}
	if ps.RequestSuccesses != 0 {
		t.Fatalf("success count should reset, got %d", ps.RequestSuccesses)
	}
	if ps.LatencyActivity != 0 {
		t.Fatalf("activity should reset, got %f", ps.LatencyActivity)
	}
	if ps.RequestTimeouts != 2 {
		t.Fatalf("timeout penalty must survive the silence reset, got %d", ps.RequestTimeouts)
	}

	// The returning peer re-bootstraps a fresh EMA while keeping its timeouts.
	s.NotifyRequestResult("peerA", 300*time.Millisecond, false)
	ps = s.GetAllPeerStats()["peerA"]
	if ps.RequestLatencyEMA != 300*time.Millisecond {
		t.Fatalf("expected fresh bootstrap after reset, got %v", ps.RequestLatencyEMA)
	}
	if ps.RequestTimeouts != 2 {
		t.Fatalf("timeout count lost after return, got %d", ps.RequestTimeouts)
	}
}

// TestRequestResultTimeoutCounting verifies that timeout=true increments
// RequestTimeouts (not RequestSuccesses) and still updates the EMA.
func TestRequestResultTimeoutCounting(t *testing.T) {
	s := New()
	s.NotifyRequestResult("peerA", 5*time.Second, true)

	ps := s.GetAllPeerStats()["peerA"]
	if ps.RequestTimeouts != 1 {
		t.Fatalf("expected RequestTimeouts=1, got %d", ps.RequestTimeouts)
	}
	if ps.RequestSuccesses != 0 {
		t.Fatalf("expected RequestSuccesses=0, got %d", ps.RequestSuccesses)
	}
	if ps.RequestLatencyEMA != 5*time.Second {
		t.Fatalf("EMA should bootstrap to timeout value, got %v", ps.RequestLatencyEMA)
	}
}

// TestRequestResultMixedCounting verifies that a mix of successes and
// timeouts increments the correct counters independently.
func TestRequestResultMixedCounting(t *testing.T) {
	s := New()
	s.NotifyRequestResult("peerA", 100*time.Millisecond, false)
	s.NotifyRequestResult("peerA", 100*time.Millisecond, false)
	s.NotifyRequestResult("peerA", 5*time.Second, true)

	ps := s.GetAllPeerStats()["peerA"]
	if ps.RequestSuccesses != 2 {
		t.Fatalf("expected RequestSuccesses=2, got %d", ps.RequestSuccesses)
	}
	if ps.RequestTimeouts != 1 {
		t.Fatalf("expected RequestTimeouts=1, got %d", ps.RequestTimeouts)
	}
}
