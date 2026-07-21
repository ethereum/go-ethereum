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
	s.NotifyRequestResult("peerA", 200*time.Millisecond, false)
	s.NotifyPeerConnect("peerA")
	if got := s.GetAllPeerStats()["peerA"].RequestLatencyEMA; got != 200*time.Millisecond {
		t.Fatalf("re-connect wiped stats: got EMA %v, want 200ms", got)
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

// TestNotifyRequestResultFirstSampleBootstrap asserts that the first
// latency sample seeds the EMA directly.
func TestNotifyRequestResultFirstSampleBootstrap(t *testing.T) {
	s := newStats("peerA")
	s.NotifyRequestResult("peerA", 200*time.Millisecond, false)

	ps := s.GetAllPeerStats()["peerA"]
	if ps.RequestLatencyEMA != 200*time.Millisecond {
		t.Fatalf("expected first sample to seed EMA at 200ms, got %v", ps.RequestLatencyEMA)
	}
}

// TestNotifyRequestResultEMAUpdate verifies the EMA formula for latency.
func TestNotifyRequestResultEMAUpdate(t *testing.T) {
	s := newStats("peerA")
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
}

// TestNotifyRequestResultSlowConvergence verifies the slow alpha
// damps convergence under sustained timeouts.
func TestNotifyRequestResultSlowConvergence(t *testing.T) {
	s := newStats("peerA")
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

// TestNotifyRequestResultIgnoresUnregisteredPeer verifies that a result for a
// peer with no entry (e.g. one that raced in after disconnect) is dropped
// rather than creating an orphan.
func TestNotifyRequestResultIgnoresUnregisteredPeer(t *testing.T) {
	s := New()
	s.NotifyRequestResult("ghost", 50*time.Millisecond, false)
	if n := len(s.GetAllPeerStats()); n != 0 {
		t.Fatalf("result for unregistered peer must not create an entry, got %d", n)
	}
}

// TestNotifyPeerDropClearsStats verifies that a dropped peer disappears
// from GetAllPeerStats.
func TestNotifyPeerDropClearsStats(t *testing.T) {
	s := newStats("peerA")
	s.NotifyRequestResult("peerA", 200*time.Millisecond, false)
	s.NotifyPeerDrop("peerA")

	if _, ok := s.GetAllPeerStats()["peerA"]; ok {
		t.Fatal("peerA stats should be removed after NotifyPeerDrop")
	}
}

// TestRequestResultIgnoredAfterDrop verifies that a late latency sample racing
// in after NotifyPeerDrop is ignored rather than recreating an orphan entry.
func TestRequestResultIgnoredAfterDrop(t *testing.T) {
	s := newStats("peerA")
	s.NotifyRequestResult("peerA", 200*time.Millisecond, false)
	s.NotifyPeerDrop("peerA")
	// Late sample racing with the drop.
	s.NotifyRequestResult("peerA", 50*time.Millisecond, false)

	if _, ok := s.GetAllPeerStats()["peerA"]; ok {
		t.Fatal("late sample after drop must not recreate the entry")
	}
}

// TestMultiplePeersIsolated verifies per-peer isolation across signal types.
func TestMultiplePeersIsolated(t *testing.T) {
	s := newStats("peerA", "peerB")
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
	s := newStats("peerA")
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
	single := newStats("peerA")
	single.NotifyRequestResult("peerA", 50*time.Millisecond, false)
	single.NotifyBlock(nil, nil)

	burst := newStats("peerA")
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
	s := newStats("peerA")
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
	s := newStats("peerA")
	for i := 0; i < 50; i++ {
		s.NotifyRequestResult("peerA", 5*time.Second, true)
		s.NotifyBlock(nil, nil)
	}
	ps := s.GetAllPeerStats()["peerA"]
	if ps.LatencyActivity != 0 {
		t.Fatalf("timeouts must not feed activity, got %f", ps.LatencyActivity)
	}
	// Timeouts still shape the latency EMA (their penalty), just not activity.
	if ps.RequestLatencyEMA != 5*time.Second {
		t.Fatalf("expected timeout EMA at 5s, got %v", ps.RequestLatencyEMA)
	}
}

// TestLatencyStateForgottenAfterSilence verifies that once a silent peer's
// activity fully decays, its fast-latency state is reset — a frozen fast EMA
// from a past active period cannot be re-armed later by rebuilding activity
// alone.
func TestLatencyStateForgottenAfterSilence(t *testing.T) {
	s := newStats("peerA")
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
	if ps.RequestLatencyEMA != 0 || ps.LatencyActivity != 0 {
		t.Fatalf("expected fast-latency state forgotten after long silence, got %+v", ps)
	}

	// A returning peer starts over: the next sample re-seeds the EMA.
	s.NotifyRequestResult("peerA", 300*time.Millisecond, false)
	if got := s.GetAllPeerStats()["peerA"].RequestLatencyEMA; got != 300*time.Millisecond {
		t.Fatalf("expected fresh bootstrap after reset, got %v", got)
	}
}

// TestTimeoutBootstrapsEMA verifies that a timeout still updates the latency
// EMA (bootstrapping it to the timeout value) even though it does not feed the
// activity gate.
func TestTimeoutBootstrapsEMA(t *testing.T) {
	s := newStats("peerA")
	s.NotifyRequestResult("peerA", 5*time.Second, true)

	ps := s.GetAllPeerStats()["peerA"]
	if ps.RequestLatencyEMA != 5*time.Second {
		t.Fatalf("EMA should bootstrap to timeout value, got %v", ps.RequestLatencyEMA)
	}
	if ps.LatencyActivity != 0 {
		t.Fatalf("a timeout must not raise activity, got %f", ps.LatencyActivity)
	}
}
