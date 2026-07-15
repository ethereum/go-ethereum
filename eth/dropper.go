// Copyright 2025 The go-ethereum Authors
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
	"cmp"
	mrand "math/rand"
	"slices"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/eth/peerstats"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
)

const (
	// Interval between peer drop events (uniform between min and max)
	peerDropIntervalMin = 3 * time.Minute
	// Interval between peer drop events (uniform between min and max)
	peerDropIntervalMax = 7 * time.Minute
	// Avoid dropping peers for some time after connection
	doNotDropBefore = 10 * time.Minute
	// How close to max should we initiate the drop timer. O should be fine,
	// dropping when no more peers can be added. Larger numbers result in more
	// aggressive drop behavior.
	peerDropThreshold = 0
	// Fraction of inbound/dialed peers to protect based on inclusion stats.
	// The top inclusionProtectionFrac of each category (by score) are
	// shielded from random dropping. 0.1 = top 10%.
	inclusionProtectionFrac = 0.1
)

var (
	// droppedInbound is the number of inbound peers dropped
	droppedInbound = metrics.NewRegisteredMeter("eth/dropper/inbound", nil)
	// droppedOutbound is the number of outbound peers dropped
	droppedOutbound = metrics.NewRegisteredMeter("eth/dropper/outbound", nil)
	// dropSkipped counts times a drop was attempted but no peer was dropped,
	// for any reason (pool has headroom, all candidates trusted/static/young,
	// or protected by inclusion stats).
	dropSkipped = metrics.NewRegisteredMeter("eth/dropper/skipped", nil)
)

// Callback type to get per-peer inclusion statistics.
type getPeerStatsFunc func() map[string]peerstats.PeerStats

// protectionCategory defines a peer scoring function and the fraction of peers
// to protect per inbound/dialed category. Multiple categories are unioned.
type protectionCategory struct {
	score func(peerstats.PeerStats) float64
	frac  float64 // fraction of max peers to protect (0.0–1.0)
}

// protectionCategories is the list of protection criteria. Each category
// independently selects its top-N peers per pool; the union is protected.
var protectionCategories = []protectionCategory{
	{func(s peerstats.PeerStats) float64 { return s.RecentFinalized }, inclusionProtectionFrac}, // Recent finalized
	{func(s peerstats.PeerStats) float64 { return s.RecentIncluded }, inclusionProtectionFrac},  // Recent included
	{func(s peerstats.PeerStats) float64 { // Request latency
		// Low-latency peers should rank higher. Peers with too few samples
		// score 0 so the existing `score > 0` filter excludes them — this
		// prevents a single lucky-fast reply from winning protection. Peers
		// whose EMA reaches the timeout also score low by this path because
		// the reciprocal of a very large duration is tiny but positive; the
		// per-pool top-N will still push faster peers ahead of them.
		if s.RequestSuccesses+s.RequestTimeouts < peerstats.MinLatencySamples {
			return 0
		}
		// Freshness gate: a peer that earned a fast EMA but then went
		// silent on announcements (no requests → no fresh samples) must
		// not keep that score indefinitely. Ignore stale data.
		if time.Since(s.LastLatencySample) > peerstats.MaxLatencyStaleness {
			return 0
		}
		if s.RequestLatencyEMA <= 0 {
			return 0
		}
		return 1.0 / float64(s.RequestLatencyEMA)
	}, inclusionProtectionFrac},
}

// dropper monitors the state of the peer pool and introduces churn by
// periodically disconnecting a random peer to make room for new connections.
// The main goal is to allow new peers to join the network and to facilitate
// continuous topology adaptation.
//
// Behavior:
//   - During sync the Downloader handles peer connections, so dropper is disabled.
//   - When not syncing and a peer category (inbound or dialed) is close to its
//     limit, a random peer from that category is disconnected every 3–7 minutes.
//   - Trusted and static peers are never dropped.
//   - Recently connected peers are also protected from dropping to give them time
//     to prove their value before being at risk of disconnection.
//   - Some peers are protected from dropping based on their contribution
//     to the tx pool. Each pool (inbound/dialed) independently selects its
//     top fraction of peers by a per-peer EMA score — a slow EMA of
//     finalized inclusions (~1-day half-life, rewards sustained long-term
//     contribution) and a fast EMA of recent block inclusions (rewards
//     current activity). The union of all protected sets is shielded from
//     random dropping, and the drop target is chosen randomly from the
//     remainder.
type dropper struct {
	maxDialPeers    int // maximum number of dialed peers
	maxInboundPeers int // maximum number of inbound peers
	peersFunc       getPeersFunc
	syncingFunc     getSyncingFunc
	peerStatsFunc   getPeerStatsFunc      // optional: inclusion stats for protection
	pruneStatsFunc  func(map[string]bool) // optional: reclaim stats for disconnected peers

	// peerDropTimer introduces churn if we are close to limit capacity.
	// We handle Dialed and Inbound connections separately
	peerDropTimer *time.Timer

	wg         sync.WaitGroup // wg for graceful shutdown
	shutdownCh chan struct{}
}

// Callback type to get the list of connected peers.
type getPeersFunc func() []*p2p.Peer

// Callback type to get syncing status.
// Returns true while syncing, false when synced.
type getSyncingFunc func() bool

func newDropper(maxDialPeers, maxInboundPeers int) *dropper {
	cm := &dropper{
		maxDialPeers:    maxDialPeers,
		maxInboundPeers: maxInboundPeers,
		peerDropTimer:   time.NewTimer(randomDuration(peerDropIntervalMin, peerDropIntervalMax)),
		shutdownCh:      make(chan struct{}),
	}
	if peerDropIntervalMin > peerDropIntervalMax {
		panic("peerDropIntervalMin duration must be less than or equal to peerDropIntervalMax duration")
	}
	return cm
}

// Start the dropper. peerStatsFunc and pruneStatsFunc are optional (nil
// disables inclusion protection and stats pruning respectively).
func (cm *dropper) Start(srv *p2p.Server, syncingFunc getSyncingFunc, peerStatsFunc getPeerStatsFunc, pruneStatsFunc func(map[string]bool)) {
	cm.peersFunc = srv.Peers
	cm.syncingFunc = syncingFunc
	cm.peerStatsFunc = peerStatsFunc
	cm.pruneStatsFunc = pruneStatsFunc
	cm.wg.Add(1)
	go cm.loop()
}

// Stop the dropper.
func (cm *dropper) Stop() {
	cm.peerDropTimer.Stop()
	close(cm.shutdownCh)
	cm.wg.Wait()
}

// dropRandomPeer selects one of the peers randomly and drops it from the peer pool.
func (cm *dropper) dropRandomPeer() bool {
	peers := cm.peersFunc()
	var numInbound int
	for _, p := range peers {
		if p.Inbound() {
			numInbound++
		}
	}
	numDialed := len(peers) - numInbound

	// Fast path: if neither pool is near capacity, every non-trusted/non-static
	// peer is already do-not-drop by pool-threshold rules. No point computing
	// inclusion protection.
	if cm.maxDialPeers-numDialed > peerDropThreshold &&
		cm.maxInboundPeers-numInbound > peerDropThreshold {
		dropSkipped.Mark(1)
		return false
	}

	// Compute the set of inclusion-protected peers before filtering.
	protected := cm.protectedPeers(peers)

	selectDoNotDrop := func(p *p2p.Peer) bool {
		return p.Trusted() || p.StaticDialed() ||
			p.Lifetime() < mclock.AbsTime(doNotDropBefore) ||
			(p.DynDialed() && cm.maxDialPeers-numDialed > peerDropThreshold) ||
			(p.Inbound() && cm.maxInboundPeers-numInbound > peerDropThreshold) ||
			protected[p]
	}

	droppable := slices.DeleteFunc(peers, selectDoNotDrop)
	if len(droppable) == 0 {
		dropSkipped.Mark(1)
		return false
	}
	p := droppable[mrand.Intn(len(droppable))]
	log.Debug("Dropping random peer", "inbound", p.Inbound(),
		"id", p.ID(), "duration", common.PrettyDuration(p.Lifetime()), "peercountbefore", len(peers))
	p.Disconnect(p2p.DiscUselessPeer)
	if p.Inbound() {
		droppedInbound.Mark(1)
	} else {
		droppedOutbound.Mark(1)
	}
	return true
}

// pruneStats reclaims stats for peers that are no longer connected. It builds
// the currently-connected id set and hands it to the stats pruner. No-op when
// pruning is disabled (nil pruneStatsFunc).
func (cm *dropper) pruneStats() {
	if cm.pruneStatsFunc == nil {
		return
	}
	peers := cm.peersFunc()
	keep := make(map[string]bool, len(peers))
	for _, p := range peers {
		keep[p.ID().String()] = true
	}
	cm.pruneStatsFunc(keep)
}

// protectedPeers computes the set of peers that should not be dropped based
// on inclusion stats. Each protection category independently selects its
// top-N peers per inbound/dialed pool; the union is returned.
func (cm *dropper) protectedPeers(peers []*p2p.Peer) map[*p2p.Peer]bool {
	if cm.peerStatsFunc == nil {
		return nil
	}
	stats := cm.peerStatsFunc()
	if len(stats) == 0 {
		return nil
	}
	// Split peers by direction.
	var inbound, dialed []*p2p.Peer
	for _, p := range peers {
		if p.Inbound() {
			inbound = append(inbound, p)
		} else {
			dialed = append(dialed, p)
		}
	}
	result := protectedPeersByPool(inbound, dialed, stats)
	if len(result) > 0 {
		log.Debug("Protecting high-value peers from drop", "protected", len(result))
	}
	return result
}

// protectedPeersByPool selects the union of top-N peers per protection
// category across the given already-split inbound and dialed pools.
// Factored from protectedPeers so tests can exercise the per-pool
// selection logic without needing to construct direction-flagged
// *p2p.Peer instances (which require unexported p2p types).
func protectedPeersByPool(inbound, dialed []*p2p.Peer, stats map[string]peerstats.PeerStats) map[*p2p.Peer]bool {
	result := make(map[*p2p.Peer]bool)
	// protectPool selects the top-frac peers from pool by score and adds them to result.
	protectPool := func(pool []*p2p.Peer, cat protectionCategory) {
		n := int(float64(len(pool)) * cat.frac)
		if n == 0 {
			return
		}
		sorted := slices.SortedFunc(slices.Values(pool), func(a, b *p2p.Peer) int {
			// descending
			scoreB := cat.score(stats[b.ID().String()])
			scoreA := cat.score(stats[a.ID().String()])
			return cmp.Compare(scoreB, scoreA)
		})
		// select top n peers excluding 0
		for _, p := range sorted[:min(n, len(sorted))] {
			if cat.score(stats[p.ID().String()]) > 0 {
				result[p] = true
			}
		}
	}
	for _, cat := range protectionCategories {
		protectPool(inbound, cat)
		protectPool(dialed, cat)
	}
	return result
}

// randomDuration generates a random duration between min and max.
func randomDuration(min, max time.Duration) time.Duration {
	if min > max {
		panic("min duration must be less than or equal to max duration")
	}
	if min == max {
		return min
	}
	return time.Duration(mrand.Int63n(int64(max-min)) + int64(min))
}

// loop is the main loop of the connection dropper.
func (cm *dropper) loop() {
	defer cm.wg.Done()

	for {
		select {
		case <-cm.peerDropTimer.C:
			// Reclaim stats entries for peers that are no longer connected,
			// covering the rare orphan left when a peer signal races with its
			// NotifyPeerDrop. Done every tick (independent of syncing) since
			// disconnects happen during sync too.
			cm.pruneStats()
			// Drop a random peer if we are not syncing and the peer count is close to the limit.
			if !cm.syncingFunc() {
				cm.dropRandomPeer()
			}
			cm.peerDropTimer.Reset(randomDuration(peerDropIntervalMin, peerDropIntervalMax))
		case <-cm.shutdownCh:
			return
		}
	}
}
