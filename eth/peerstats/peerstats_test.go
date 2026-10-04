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

// TestPeerLifecycle verifies that an entry exists only between
// NotifyPeerConnect and NotifyPeerDrop: connecting creates it, reconnecting
// keeps its stats, dropping removes it, and signals for a peer without an
// entry never create one, so a peer that has disconnected cannot be
// resurrected by a late signal.
func TestPeerLifecycle(t *testing.T) {
	s := New()

	// Signals for a peer that never connected are ignored.
	s.NotifyBlock(map[string]int{"ghost": 3}, map[string]int{"ghost": 5})
	s.NotifyRequestResult("ghost", 50*time.Millisecond, false)
	if n := len(s.GetAllPeerStats()); n != 0 {
		t.Fatalf("signals for unregistered peer must not create entries, got %d", n)
	}

	// Connecting creates an entry; reconnecting keeps its stats.
	s.NotifyPeerConnect("peerA")
	if _, ok := s.GetAllPeerStats()["peerA"]; !ok {
		t.Fatal("expected peerA entry after connect")
	}
	s.NotifyRequestResult("peerA", 200*time.Millisecond, false)
	s.NotifyPeerConnect("peerA")
	if got := s.GetAllPeerStats()["peerA"].RequestLatencyEMA; got != 200*time.Millisecond {
		t.Fatalf("re-connect wiped stats: got EMA %v, want 200ms", got)
	}

	// Dropping removes the entry, and a late signal does not recreate it.
	s.NotifyPeerDrop("peerA")
	if _, ok := s.GetAllPeerStats()["peerA"]; ok {
		t.Fatal("NotifyPeerDrop should remove the entry")
	}
	s.NotifyBlock(map[string]int{"peerA": 1}, map[string]int{"peerA": 10})
	s.NotifyRequestResult("peerA", 50*time.Millisecond, false)
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

// TestNotifyRequestResultEMAUpdate verifies the latency EMA: the first sample
// seeds it directly, later samples blend in with latencyEMAAlpha.
func TestNotifyRequestResultEMAUpdate(t *testing.T) {
	s := newStats("peerA")
	s.NotifyRequestResult("peerA", 100*time.Millisecond, false)
	if got := s.GetAllPeerStats()["peerA"].RequestLatencyEMA; got != 100*time.Millisecond {
		t.Fatalf("first sample should seed the EMA: got %v, want 100ms", got)
	}
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

// TestTimeoutDoesNotFeedActivity verifies that timeouts update the latency EMA
// but never contribute to the activity rate — a peer cannot become
// protection-eligible by timing out. A timeout-only peer also keeps its EMA:
// the silence reset only applies to peers that earned a fast one.
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
