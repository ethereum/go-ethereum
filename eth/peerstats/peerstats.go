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

// Package peerstats maintains per-peer quality metrics used by the peer
// dropper to protect high-value peers from random disconnection.
//
// The package is a passive accumulator: it exposes entry points for its
// signal producers (txtracker for inclusion/finalization, the tx fetcher
// for latency, the handler for peer-drop cleanup) and a read-only
// snapshot for its consumer (the dropper). It has no goroutine of its
// own — all mutation is serialized by a single mutex.
//
// Signal sources:
//   - NotifyBlock(inclusions, finalized) — per-block deltas from txtracker
//     (computed under txtracker's own lock, then passed in after release)
//   - NotifyRequestLatency(peer, latency) — per-request samples from the
//     fetcher; timeouts are reported with the timeout value so slow peers
//     contribute to the EMA
//   - NotifyPeerDrop(peer) — called from the handler on disconnect
package peerstats

import (
	"sync"
	"time"
)

const (
	// EMA smoothing factor for per-block inclusion rate.
	emaAlpha = 0.05
	// EMA smoothing factor for per-block finalization rate. Very slow on
	// purpose: finalization is permanent, and the score should reflect
	// sustained contribution over long windows, not recent bursts.
	// Half-life ≈ 6930 chain heads (~23 hours on 12s blocks).
	finalizedEMAAlpha = 0.0001
	// EMA smoothing factor for per-request latency average. Slow on purpose:
	// short bursts shouldn't shift the score, sustained behavior should.
	// Half-life ≈ ln(0.5)/ln(0.99) ≈ 69 samples.
	latencyEMAAlpha = 0.01
	// MinLatencySamples is the number of latency samples a peer must accumulate
	// before its RequestLatencyEMA is considered meaningful for protection.
	// Prevents a single lucky-fast reply from displacing established peers.
	MinLatencySamples = 100
	// MaxLatencyStaleness is the oldest allowed age of a peer's last
	// latency sample before their RequestLatencyEMA is disregarded for
	// protection. Prevents a peer from earning a fast score during a
	// burst of activity and then holding protection indefinitely by
	// going silent on tx announcements (no further requests → no fresh
	// samples → EMA frozen at its last value).
	MaxLatencyStaleness = 10 * time.Minute
)

// PeerStats is the exported per-peer snapshot returned by GetAllPeerStats.
type PeerStats struct {
	RecentFinalized   float64       // EMA of per-block finalization credits (slow)
	RecentIncluded    float64       // EMA of per-block inclusions (fast)
	RequestLatencyEMA time.Duration // Slow EMA of tx-request response latency (timeouts count as the timeout value)
	RequestSamples    int64         // Number of latency samples seen (for bootstrap guard)
	LastLatencySample time.Time     // Wall-clock time of the most recent latency sample (for staleness gate)
}

// peerStats is the internal mutable state per peer.
type peerStats struct {
	recentFinalized   float64
	recentIncluded    float64
	requestLatencyEMA time.Duration
	requestSamples    int64
	lastLatencySample time.Time
}

// Stats is the per-peer quality aggregator.
type Stats struct {
	mu    sync.Mutex
	peers map[string]*peerStats
}

// New creates an empty Stats.
func New() *Stats {
	return &Stats{peers: make(map[string]*peerStats)}
}

// NotifyBlock ingests a per-block update. `inclusions` is the count of the head
// block's transactions attributed to each peer; peers with a positive
// count get a stats entry created if one doesn't exist (this is how
// peerstats learns about newly-active peers). Peers not in the map but
// already tracked have their EMA decay with a zero sample.
//
// `finalized` is per-peer credits accumulated since the last NotifyBlock;
// credits are only applied to peers already tracked — we don't resurrect
// dropped peers from historical finalization data.
//
// NotifyBlock must NOT be called while the caller holds any other lock that
// could be acquired by peerstats callers in reverse order. Current callers
// (txtracker.handleChainHead) release their lock before invoking NotifyBlock.
func (s *Stats) NotifyBlock(inclusions, finalized map[string]int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure a stats entry exists for any peer that just had an inclusion.
	// This is the primary path by which peerstats learns about a peer's
	// inclusion activity.
	for peer, count := range inclusions {
		if count > 0 && s.peers[peer] == nil {
			s.peers[peer] = &peerStats{}
		}
	}
	// Update inclusion and finalization EMAs for every tracked peer. A
	// peer not present in the respective delta map gets a 0 contribution
	// — pure decay. Finalization credits for peers no longer tracked are
	// ignored (don't resurrect dropped peers from historical data).
	for peer, ps := range s.peers {
		ps.recentIncluded = (1-emaAlpha)*ps.recentIncluded + emaAlpha*float64(inclusions[peer])
		ps.recentFinalized = (1-finalizedEMAAlpha)*ps.recentFinalized + finalizedEMAAlpha*float64(finalized[peer])
	}
}

// NotifyRequestLatency records a tx-request response latency sample for
// the given peer. Timeouts should be reported as the timeout value.
// Creates a peer entry if one doesn't exist (a peer may have latency
// samples before any inclusion signal).
func (s *Stats) NotifyRequestLatency(peer string, latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ps := s.peers[peer]
	if ps == nil {
		ps = &peerStats{}
		s.peers[peer] = ps
	}
	if ps.requestSamples == 0 {
		// Bootstrap the EMA with the first sample so it doesn't drift up
		// from zero over many samples before reaching realistic values.
		ps.requestLatencyEMA = latency
	} else {
		ps.requestLatencyEMA = time.Duration(
			float64(ps.requestLatencyEMA)*(1-latencyEMAAlpha) +
				float64(latency)*latencyEMAAlpha,
		)
	}
	ps.requestSamples++
	ps.lastLatencySample = time.Now()
}

// NotifyPeerDrop removes a peer's stats on disconnect. A rare stale
// latency sample racing with the drop may recreate the peer entry with
// one sample; that entry can never earn protection (MinLatencySamples
// guard) and is harmless.
func (s *Stats) NotifyPeerDrop(peer string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.peers, peer)
}

// GetAllPeerStats returns a snapshot of per-peer stats. Called by the
// dropper every few minutes; allocation cost is negligible at that rate.
func (s *Stats) GetAllPeerStats() map[string]PeerStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make(map[string]PeerStats, len(s.peers))
	for id, ps := range s.peers {
		result[id] = PeerStats{
			RecentFinalized:   ps.recentFinalized,
			RecentIncluded:    ps.recentIncluded,
			RequestLatencyEMA: ps.requestLatencyEMA,
			RequestSamples:    ps.requestSamples,
			LastLatencySample: ps.lastLatencySample,
		}
	}
	return result
}
