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
//   - NotifyRequestResult(peer, latency, timeout) — per-request outcomes
//     from the fetcher; timeouts are reported with the timeout value so
//     slow peers contribute to the EMA. Non-timeout results are only
//     reported for deliveries with pool-accepted txs, and only those feed
//     the activity rate gating latency protection
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
	// EMA smoothing factor for the per-block latency-sample activity rate.
	// Half-life ≈ 50 chain heads (~10 minutes on 12s blocks): eligibility
	// for latency protection must be continuously maintained at roughly
	// this cadence, it cannot be front-loaded in a burst and then held.
	latencyActivityAlpha = 0.014
	// MinLatencyActivity is the minimum sustained rate of accepted-delivery
	// latency samples (per block, EMA-smoothed) a peer must maintain for its
	// RequestLatencyEMA to be considered for protection. 0.2 ≈ one accepted
	// fetch per five blocks (~1/minute). Replaces both an absolute sample
	// count (front-loadable) and a last-sample staleness check (maintainable
	// with one sample per window): a decaying rate expires on its own and
	// demands sustained useful work.
	MinLatencyActivity = 0.2
	// latencyResetThreshold is the activity level below which a peer's
	// latency state is forgotten entirely. Without this, a peer could
	// earn a fast EMA, go silent (activity decays, eligibility lost) and
	// later re-arm the frozen EMA by rebuilding activity alone. Once
	// activity has decayed this far (~75 minutes of silence from the
	// eligibility threshold), the peer starts over as a stranger.
	latencyResetThreshold = 0.001
)

// PeerStats is the exported per-peer snapshot returned by GetAllPeerStats.
type PeerStats struct {
	RecentFinalized   float64       // EMA of per-block finalization credits (slow)
	RecentIncluded    float64       // EMA of per-block inclusions (fast)
	RequestLatencyEMA time.Duration // Slow EMA of tx-request response latency (timeouts count as the timeout value)
	RequestSuccesses  int64         // Accepted deliveries (requests answered in time with ≥1 pool-accepted tx)
	RequestTimeouts   int64         // Requests that timed out
	LatencyActivity   float64       // EMA of accepted-delivery samples per block (eligibility gate for latency protection)
}

// peerStats is the internal mutable state per peer.
type peerStats struct {
	recentFinalized   float64
	recentIncluded    float64
	requestLatencyEMA time.Duration
	requestSuccesses  int64
	requestTimeouts   int64
	latencyActivity   float64
	pendingSamples    int // accepted-delivery samples since the last NotifyBlock
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

		// Fold the accepted-delivery samples gathered since the previous
		// head into the activity rate, then let it decay like the other
		// per-block EMAs.
		ps.latencyActivity = (1-latencyActivityAlpha)*ps.latencyActivity + latencyActivityAlpha*float64(ps.pendingSamples)
		ps.pendingSamples = 0

		// A peer silent long enough for its activity to fully decay
		// forgets its latency history: a frozen fast EMA from a past
		// active period must not be re-armable by rebuilding activity
		// alone (see latencyResetThreshold). Gated on success history —
		// only successes create a fast EMA worth forgetting; a
		// timeout-only peer (activity permanently zero) keeps its
		// penalty record.
		if ps.latencyActivity < latencyResetThreshold && ps.requestSuccesses != 0 {
			ps.latencyActivity = 0
			ps.requestLatencyEMA = 0
			ps.requestSuccesses = 0
			ps.requestTimeouts = 0
		}
	}
}

// NotifyRequestResult records a tx-request outcome for the given peer.
// latency is the round-trip time (for timeouts, pass the timeout value).
// timeout indicates whether the request timed out rather than receiving an
// accepted delivery (the fetcher only reports non-timeout results for
// deliveries with ≥1 pool-accepted tx). Both cases update the latency EMA;
// only accepted deliveries feed the activity rate that gates protection —
// a peer cannot become protection-eligible by timing out, and penalties
// remain ungated. Creates a peer entry if one doesn't exist.
func (s *Stats) NotifyRequestResult(peer string, latency time.Duration, timeout bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ps := s.peers[peer]
	if ps == nil {
		ps = &peerStats{}
		s.peers[peer] = ps
	}
	if ps.requestSuccesses+ps.requestTimeouts == 0 {
		// Bootstrap the EMA with the first sample so it doesn't drift up
		// from zero over many samples before reaching realistic values.
		ps.requestLatencyEMA = latency
	} else {
		ps.requestLatencyEMA = time.Duration(
			float64(ps.requestLatencyEMA)*(1-latencyEMAAlpha) +
				float64(latency)*latencyEMAAlpha,
		)
	}
	if timeout {
		ps.requestTimeouts++
	} else {
		ps.requestSuccesses++
		ps.pendingSamples++
	}
}

// NotifyPeerDrop removes a peer's stats on disconnect.
//
// A signal (NotifyRequestResult or NotifyBlock) for the same peer can race
// with the drop and land just after this deletion, recreating an orphan
// entry that no future NotifyPeerDrop will ever clean. Such orphans are
// never read — the dropper only looks up currently-connected peers — but
// left alone they accumulate for the node's lifetime. Prune reclaims them.
func (s *Stats) NotifyPeerDrop(peer string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.peers, peer)
}

// Prune removes stats for every peer not present in keep. The dropper calls
// this periodically with the set of currently-connected peer IDs to reclaim
// orphan entries left by a signal that raced with NotifyPeerDrop (see there).
// Pruning a still-connected peer that only just gained an entry is harmless:
// it resets a handful of early samples that self-heal on the peer's next
// activity, and such a peer cannot yet meet the protection thresholds.
func (s *Stats) Prune(keep map[string]bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id := range s.peers {
		if !keep[id] {
			delete(s.peers, id)
		}
	}
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
			RequestSuccesses:  ps.requestSuccesses,
			RequestTimeouts:   ps.requestTimeouts,
			LatencyActivity:   ps.latencyActivity,
		}
	}
	return result
}
