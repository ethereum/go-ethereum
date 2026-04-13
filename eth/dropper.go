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
	// dropSkipped counts times a drop was skipped because all
	// droppable candidates were protected by inclusion stats.
	dropSkipped = metrics.NewRegisteredMeter("eth/dropper/protected", nil)
)

// PeerInclusionStats holds the per-peer inclusion data needed by the dropper
// to decide which peers to protect. Any stats provider (e.g. txtracker) can
// implement getPeerInclusionStatsFunc by returning this struct per peer ID.
type PeerInclusionStats struct {
	Finalized      int64   // Cumulative finalized inclusions attributed to this peer
	RecentIncluded float64 // EMA of per-block inclusions (0 if not tracked)
}

// Callback type to get per-peer inclusion statistics.
type getPeerInclusionStatsFunc func() map[string]PeerInclusionStats

// protectionCategory defines a peer scoring function and the fraction of peers
// to protect per inbound/dialed category. Multiple categories are unioned.
type protectionCategory struct {
	name  string
	score func(PeerInclusionStats) float64
	frac  float64 // fraction of max peers to protect (0.0–1.0)
}

// protectionCategories is the list of protection criteria. Each category
// independently selects its top-N peers per pool; the union is protected.
var protectionCategories = []protectionCategory{
	{"total-finalized", func(s PeerInclusionStats) float64 { return float64(s.Finalized) }, inclusionProtectionFrac},
	{"recent-included", func(s PeerInclusionStats) float64 { return s.RecentIncluded }, inclusionProtectionFrac},
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
//   - Some peers are protected from dropping based on their role. This is not based
//     on a unified score function, but rather on the concept of protected peer pools.
//     Each pool independently selects its top fraction of peers by a specific score
//     (e.g. total finalized inclusions, recent inclusion EMA); the union of all
//     protected sets is shielded from random dropping, and the drop target is chosen
//     randomly from the remainder.
type dropper struct {
	maxDialPeers    int // maximum number of dialed peers
	maxInboundPeers int // maximum number of inbound peers
	peersFunc       getPeersFunc
	syncingFunc     getSyncingFunc
	peerStatsFunc   getPeerInclusionStatsFunc // optional: inclusion stats for protection

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

// Start the dropper. peerStatsFunc is optional (nil disables inclusion
// protection).
func (cm *dropper) Start(srv *p2p.Server, syncingFunc getSyncingFunc, peerStatsFunc getPeerInclusionStatsFunc) {
	cm.peersFunc = srv.Peers
	cm.syncingFunc = syncingFunc
	cm.peerStatsFunc = peerStatsFunc
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
		if len(protected) > 0 {
			dropSkipped.Mark(1)
		}
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
	// protectPool selects the top-frac peers from pool by score and adds them to result.
	result := make(map[*p2p.Peer]bool)
	protectPool := func(pool []*p2p.Peer, score func(*p2p.Peer) float64, frac float64) {
		n := int(float64(len(pool)) * frac)
		if n == 0 {
			return
		}
		sorted := slices.SortedFunc(slices.Values(pool), func(a, b *p2p.Peer) int {
			return cmp.Compare(score(b), score(a)) // descending
		})
		top := slices.DeleteFunc(sorted[:min(n, len(sorted))], func(p *p2p.Peer) bool {
			return score(p) <= 0
		})
		for _, p := range top {
			result[p] = true
		}
	}
	for _, cat := range protectionCategories {
		score := func(p *p2p.Peer) float64 {
			return cat.score(stats[p.ID().String()])
		}
		protectPool(inbound, score, cat.frac)
		protectPool(dialed, score, cat.frac)
	}
	if len(result) > 0 {
		log.Debug("Protecting high-value peers from drop", "protected", len(result))
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
