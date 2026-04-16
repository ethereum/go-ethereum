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

package eth

import (
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/eth/peerstats"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

func makePeers(n int) []*p2p.Peer {
	peers := make([]*p2p.Peer, n)
	for i := range peers {
		id := enode.ID{byte(i)}
		peers[i] = p2p.NewPeer(id, fmt.Sprintf("peer%d", i), nil)
	}
	return peers
}

func TestProtectedPeersNoStats(t *testing.T) {
	cm := &dropper{maxDialPeers: 20, maxInboundPeers: 30}
	cm.peerStatsFunc = func() map[string]peerstats.PeerStats { return nil }

	peers := makePeers(10)
	protected := cm.protectedPeers(peers)
	if len(protected) != 0 {
		t.Fatalf("expected no protected peers with nil stats, got %d", len(protected))
	}
}

func TestProtectedPeersEmptyStats(t *testing.T) {
	cm := &dropper{maxDialPeers: 20, maxInboundPeers: 30}
	cm.peerStatsFunc = func() map[string]peerstats.PeerStats {
		return map[string]peerstats.PeerStats{}
	}

	peers := makePeers(10)
	protected := cm.protectedPeers(peers)
	if len(protected) != 0 {
		t.Fatalf("expected no protected peers with empty stats, got %d", len(protected))
	}
}

func TestProtectedPeersTopPeer(t *testing.T) {
	// 20 peers, 10% of 20 = 2 protected per category.
	cm := &dropper{maxDialPeers: 20, maxInboundPeers: 30}

	peers := makePeers(20)
	stats := make(map[string]peerstats.PeerStats)
	stats[peers[0].ID().String()] = peerstats.PeerStats{RecentFinalized: 100}
	stats[peers[1].ID().String()] = peerstats.PeerStats{RecentIncluded: 5.0}

	cm.peerStatsFunc = func() map[string]peerstats.PeerStats { return stats }

	protected := cm.protectedPeers(peers)
	if len(protected) != 2 {
		t.Fatalf("expected 2 protected peers, got %d", len(protected))
	}
	if !protected[peers[0]] {
		t.Fatal("peer 0 should be protected (top RecentFinalized)")
	}
	if !protected[peers[1]] {
		t.Fatal("peer 1 should be protected (top RecentIncluded)")
	}
}

func TestProtectedPeersZeroScore(t *testing.T) {
	cm := &dropper{maxDialPeers: 20, maxInboundPeers: 30}

	peers := makePeers(10)
	stats := make(map[string]peerstats.PeerStats)
	for _, p := range peers {
		stats[p.ID().String()] = peerstats.PeerStats{}
	}
	cm.peerStatsFunc = func() map[string]peerstats.PeerStats { return stats }

	protected := cm.protectedPeers(peers)
	if len(protected) != 0 {
		t.Fatalf("expected no protection with zero scores, got %d", len(protected))
	}
}

func TestProtectedPeersOverlap(t *testing.T) {
	// One peer is top in both categories — counted once.
	cm := &dropper{maxDialPeers: 20, maxInboundPeers: 30}

	peers := makePeers(20)
	stats := make(map[string]peerstats.PeerStats)
	stats[peers[0].ID().String()] = peerstats.PeerStats{RecentFinalized: 100, RecentIncluded: 5.0}

	cm.peerStatsFunc = func() map[string]peerstats.PeerStats { return stats }

	protected := cm.protectedPeers(peers)
	if len(protected) != 1 {
		t.Fatalf("expected 1 protected peer (overlap), got %d", len(protected))
	}
}

func TestProtectedPeersNilFunc(t *testing.T) {
	cm := &dropper{maxDialPeers: 20, maxInboundPeers: 30}
	// peerStatsFunc is nil (default).

	peers := makePeers(10)
	protected := cm.protectedPeers(peers)
	if protected != nil {
		t.Fatalf("expected nil with nil stats func, got %v", protected)
	}
}

// TestProtectedByPoolPerPoolTopN verifies that the top-N selection runs
// independently in each of the inbound and dialed pools, not globally.
// With 10 peers per pool and inclusionProtectionFrac=0.1, exactly 1 peer
// is protected per pool per category — so 2 total (one per pool), both
// for the RecentFinalized category since we don't set RecentIncluded.
func TestProtectedByPoolPerPoolTopN(t *testing.T) {
	inbound := makePeers(10)
	dialed := makePeers(10)
	// Distinguish dialed peer IDs from inbound so stats maps don't collide.
	for i := range dialed {
		id := enode.ID{byte(100 + i)}
		dialed[i] = p2p.NewPeer(id, fmt.Sprintf("dialed%d", i), nil)
	}
	// Strictly increasing scores: highest wins in each pool.
	stats := make(map[string]peerstats.PeerStats)
	for i, p := range inbound {
		stats[p.ID().String()] = peerstats.PeerStats{RecentFinalized: float64(1 + i)}
	}
	for i, p := range dialed {
		stats[p.ID().String()] = peerstats.PeerStats{RecentFinalized: float64(1 + i)}
	}

	protected := protectedPeersByPool(inbound, dialed, stats)

	// Expect top 1 of inbound (inbound[9]) and top 1 of dialed (dialed[9]).
	if len(protected) != 2 {
		t.Fatalf("expected 2 protected peers (1 per pool), got %d", len(protected))
	}
	if !protected[inbound[9]] {
		t.Error("expected top inbound peer to be protected")
	}
	if !protected[dialed[9]] {
		t.Error("expected top dialed peer to be protected")
	}
}

// TestProtectedByPoolCrossCategoryOverlap verifies that the union across
// protection categories is correctly deduplicated: a peer that wins in
// multiple categories appears once, and category winners are all
// protected. Uses a pool large enough that frac*len yields n=2 per
// category, so cross-category overlap is observable.
func TestProtectedByPoolCrossCategoryOverlap(t *testing.T) {
	// 20 dialed peers so 0.1 * 20 = 2 protected per category.
	dialed := makePeers(20)
	// P0: high RecentFinalized only. P1: high RecentIncluded only. P2: high both.
	// With n=2 per category:
	//   RecentFinalized winners: P2 (tie-broken-ok), P0
	//   RecentIncluded winners: P2, P1
	// Union: {P0, P1, P2}.
	stats := make(map[string]peerstats.PeerStats)
	stats[dialed[0].ID().String()] = peerstats.PeerStats{RecentFinalized: 100, RecentIncluded: 0}
	stats[dialed[1].ID().String()] = peerstats.PeerStats{RecentFinalized: 0, RecentIncluded: 5.0}
	stats[dialed[2].ID().String()] = peerstats.PeerStats{RecentFinalized: 200, RecentIncluded: 10.0}

	protected := protectedPeersByPool(nil, dialed, stats)

	if len(protected) != 3 {
		t.Fatalf("expected 3 protected peers (union of category winners), got %d", len(protected))
	}
	for _, idx := range []int{0, 1, 2} {
		if !protected[dialed[idx]] {
			t.Errorf("peer %d should be protected", idx)
		}
	}
}

// TestProtectedByPoolPerPoolIndependence locks in that selection runs
// per-pool, not globally. Every inbound peer scores higher than every
// dialed peer, so a global top-N would pick only inbound peers. Per-pool
// top-N must still protect the top dialed peers.
func TestProtectedByPoolPerPoolIndependence(t *testing.T) {
	// 20 inbound, 20 dialed — frac=0.1 → 2 protected per pool per category.
	// Global top-4 of RecentFinalized would be inbound[16..19] — zero dialed.
	inbound := makePeers(20)
	dialed := make([]*p2p.Peer, 20)
	for i := range dialed {
		id := enode.ID{byte(100 + i)}
		dialed[i] = p2p.NewPeer(id, fmt.Sprintf("dialed%d", i), nil)
	}
	stats := make(map[string]peerstats.PeerStats)
	// Every inbound peer outscores every dialed peer.
	for i, p := range inbound {
		stats[p.ID().String()] = peerstats.PeerStats{RecentFinalized: float64(1000 + i)}
	}
	for i, p := range dialed {
		stats[p.ID().String()] = peerstats.PeerStats{RecentFinalized: float64(1 + i)}
	}

	protected := protectedPeersByPool(inbound, dialed, stats)

	// Per-pool top-2 of RecentFinalized:
	//   inbound: inbound[18], inbound[19]
	//   dialed:  dialed[18], dialed[19]
	// Global top-N would contain zero dialed peers, so asserting the top
	// dialed peers are protected enforces per-pool independence.
	if !protected[dialed[19]] {
		t.Fatal("top dialed peer must be protected regardless of globally-higher inbound peers")
	}
	if !protected[dialed[18]] {
		t.Fatal("second-top dialed peer must be protected regardless of globally-higher inbound peers")
	}
	if !protected[inbound[19]] || !protected[inbound[18]] {
		t.Fatal("top inbound peers must also be protected")
	}
	if len(protected) != 4 {
		t.Fatalf("expected 4 protected peers (top-2 of each pool), got %d", len(protected))
	}
}

// TestProtectedByPoolRequestLatencyBasic verifies the latency protection
// category: with no competing inclusion stats, the lowest-latency peers
// (among those with enough samples) win top-N protection.
func TestProtectedByPoolRequestLatencyBasic(t *testing.T) {
	dialed := makePeers(20) // frac=0.1 → n=2 per category
	stats := make(map[string]peerstats.PeerStats)
	// Three peers have enough samples; the two fastest should win.
	stats[dialed[0].ID().String()] = peerstats.PeerStats{
		RequestLatencyEMA: 50 * time.Millisecond,
		RequestSuccesses:    peerstats.MinLatencySamples,
		LastLatencySample: time.Now(),
	}
	stats[dialed[1].ID().String()] = peerstats.PeerStats{
		RequestLatencyEMA: 100 * time.Millisecond,
		RequestSuccesses:    peerstats.MinLatencySamples,
		LastLatencySample: time.Now(),
	}
	stats[dialed[2].ID().String()] = peerstats.PeerStats{
		RequestLatencyEMA: 2 * time.Second,
		RequestSuccesses:    peerstats.MinLatencySamples,
		LastLatencySample: time.Now(),
	}

	protected := protectedPeersByPool(nil, dialed, stats)

	if !protected[dialed[0]] {
		t.Error("fastest peer should be protected")
	}
	if !protected[dialed[1]] {
		t.Error("second-fastest peer should be protected")
	}
	if protected[dialed[2]] {
		t.Error("slowest peer should not be in top-2")
	}
	if len(protected) != 2 {
		t.Fatalf("expected top-2 latency protection, got %d", len(protected))
	}
}

// TestProtectedByPoolRequestLatencyBootstrapGuard verifies that peers with
// fewer than MinLatencySamples do not earn latency-based protection, even
// if their few samples indicate very low latency.
func TestProtectedByPoolRequestLatencyBootstrapGuard(t *testing.T) {
	dialed := makePeers(20)
	stats := make(map[string]peerstats.PeerStats)
	// A lucky-fast peer with only 1 sample — must NOT be protected.
	stats[dialed[0].ID().String()] = peerstats.PeerStats{
		RequestLatencyEMA: 1 * time.Millisecond,
		RequestSuccesses:    1,
	}
	// A warmed-up but slower peer — should be protected on latency.
	stats[dialed[1].ID().String()] = peerstats.PeerStats{
		RequestLatencyEMA: 500 * time.Millisecond,
		RequestSuccesses:    peerstats.MinLatencySamples,
		LastLatencySample: time.Now(),
	}

	protected := protectedPeersByPool(nil, dialed, stats)

	if protected[dialed[0]] {
		t.Error("under-sampled peer should not be protected (bootstrap guard)")
	}
	if !protected[dialed[1]] {
		t.Error("warmed-up peer should be protected")
	}
}

// TestProtectedByPoolRequestLatencyPerPool verifies that the latency
// category selects top-N per pool independently, consistent with the
// other categories. An inbound peer with lower latency does not prevent
// a dialed peer from being protected as top of the dialed pool.
func TestProtectedByPoolRequestLatencyPerPool(t *testing.T) {
	inbound := makePeers(20)
	dialed := make([]*p2p.Peer, 20)
	for i := range dialed {
		id := enode.ID{byte(100 + i)}
		dialed[i] = p2p.NewPeer(id, fmt.Sprintf("dialed%d", i), nil)
	}
	stats := make(map[string]peerstats.PeerStats)
	// All inbound peers are very fast (50ms).
	for _, p := range inbound {
		stats[p.ID().String()] = peerstats.PeerStats{
			RequestLatencyEMA: 50 * time.Millisecond,
			RequestSuccesses:    peerstats.MinLatencySamples,
			LastLatencySample: time.Now(),
		}
	}
	// Dialed peers are slower (1s) — globally they would all lose, but
	// per-pool top-N should still protect two of them.
	for _, p := range dialed {
		stats[p.ID().String()] = peerstats.PeerStats{
			RequestLatencyEMA: 1 * time.Second,
			RequestSuccesses:    peerstats.MinLatencySamples,
			LastLatencySample: time.Now(),
		}
	}

	protected := protectedPeersByPool(inbound, dialed, stats)

	// 2 from inbound + 2 from dialed = 4.
	var dialedProtected int
	for _, p := range dialed {
		if protected[p] {
			dialedProtected++
		}
	}
	if dialedProtected != 2 {
		t.Fatalf("expected 2 dialed peers protected by per-pool top-N, got %d", dialedProtected)
	}
}

// TestProtectedByPoolRequestLatencyStale verifies that the freshness gate
// excludes peers whose latency EMA is valid (meeting the sample count and
// fast value) but whose last sample is older than MaxLatencyStaleness.
// A peer cannot serve a burst of fast replies, go silent on announcements,
// and keep latency-based protection indefinitely.
func TestProtectedByPoolRequestLatencyStale(t *testing.T) {
	dialed := makePeers(20)
	stats := make(map[string]peerstats.PeerStats)
	// Fresh, fast peer — should be protected.
	stats[dialed[0].ID().String()] = peerstats.PeerStats{
		RequestLatencyEMA: 50 * time.Millisecond,
		RequestSuccesses:    peerstats.MinLatencySamples,
		LastLatencySample: time.Now(),
	}
	// Stale, fast peer — was fast, but hasn't answered in too long.
	// Same EMA and sample count as the fresh peer; only staleness differs.
	stats[dialed[1].ID().String()] = peerstats.PeerStats{
		RequestLatencyEMA: 50 * time.Millisecond,
		RequestSuccesses:    peerstats.MinLatencySamples,
		LastLatencySample: time.Now().Add(-2 * peerstats.MaxLatencyStaleness),
	}

	protected := protectedPeersByPool(nil, dialed, stats)

	if !protected[dialed[0]] {
		t.Error("fresh fast peer must be protected")
	}
	if protected[dialed[1]] {
		t.Error("stale peer must NOT keep latency protection despite fast EMA")
	}
}
