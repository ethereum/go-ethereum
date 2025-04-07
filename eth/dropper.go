// Copyright 2015 The go-ethereum Authors
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
	crand "crypto/rand"
	"encoding/binary"
	mrand "math/rand"
	"slices"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
)

const (
	// Interval between peer drop events (uniform between min and max)
	peerDropIntervalMin = 3 * time.Minute
	// Interval between peer drop events (uniform between min and max)
	peerDropIntervalMax = 7 * time.Minute
	// Avoid dropping peers for some time after connection
	doNotDropBefore = 2 * peerDropIntervalMax
	// How close to max should we initiate the drop timer. O should be fine,
	// dropping when no more peers can be added. Larger numbers result in more
	// aggressive drop behavior.
	peerDropThreshold = 0
	// Sync status poll interval (no need to be too reactive here)
	syncCheckInterval = 60 * time.Second
)

// dropper monitors the state of the peer pool and makes changes as follows:
//   - during sync the Downloader handles peer connections, so dropper is disabled
//   - if not syncing and the peer count is close to the limit, it drops peers
//     randomly every peerDropInterval to make space for new peers
//   - peers are dropped separately from the inboud pool and from the dialed pool
type dropper struct {
	dropperConfig
	peersFunc   getPeersFunc
	syncingFunc getSyncingFunc

	// The peerDrop timers introduce churn if we are close to limit capacity.
	// We handle Dialed and Inbound connections separately
	peerDropDialedTimer  *mclock.Alarm
	peerDropInboundTimer *mclock.Alarm

	peerEventCh chan *p2p.PeerEvent // channel for peer event changes
	sub         event.Subscription  // subscription to peerEventCh

	wg         sync.WaitGroup // wg for graceful shutdown
	shutdownCh chan struct{}
}

// Callback type to get the list of connected peers.
type getPeersFunc func() []*p2p.Peer

// Callback type to get syncing status.
// Returns true while syncing, false when synced.
type getSyncingFunc func() bool

type dropperConfig struct {
	maxDialPeers    int // maximum number of dialed peers
	maxInboundPeers int // maximum number of inbound peers
	log             log.Logger
	clock           mclock.Clock
	rand            *mrand.Rand
}

func (cfg dropperConfig) withDefaults() dropperConfig {
	if cfg.log == nil {
		cfg.log = log.Root()
	}
	if cfg.clock == nil {
		cfg.clock = mclock.System{}
	}
	if cfg.rand == nil {
		seedb := make([]byte, 8)
		crand.Read(seedb)
		seed := int64(binary.BigEndian.Uint64(seedb))
		cfg.rand = mrand.New(mrand.NewSource(seed))
	}
	return cfg
}

func newDropper(config *dropperConfig) *dropper {
	cfg := config.withDefaults()
	cm := &dropper{
		dropperConfig:        cfg,
		peerDropDialedTimer:  mclock.NewAlarm(cfg.clock),
		peerDropInboundTimer: mclock.NewAlarm(cfg.clock),
		peerEventCh:          make(chan *p2p.PeerEvent),
		shutdownCh:           make(chan struct{}),
	}
	if peerDropIntervalMin > peerDropIntervalMax {
		panic("peerDropIntervalMin duration must be less than or equal to peerDropIntervalMax duration")
	}
	cm.log.Info("New Dropper", "maxDialPeers", cm.maxDialPeers, "threshold", peerDropThreshold,
		"intervalMin", peerDropIntervalMin, "intervalMax", peerDropIntervalMax)
	return cm
}

// Start the dropper.
func (cm *dropper) Start(srv *p2p.Server, syncingFunc getSyncingFunc) {
	cm.wg.Add(1)
	cm.peersFunc = srv.Peers
	cm.syncingFunc = syncingFunc
	cm.sub = srv.SubscribeEvents(cm.peerEventCh)
	go cm.loop()
}

// Stop the dropper.
func (cm *dropper) Stop() {
	cm.sub.Unsubscribe()
	cm.peerDropInboundTimer.Stop()
	cm.peerDropDialedTimer.Stop()
	close(cm.shutdownCh)
	cm.wg.Wait()
}

// numPeers returns the current number of peers and its breakdown (dialed or inbound).
func (cm *dropper) numPeers() (numPeers int, numDialed int, numInbound int) {
	peers := cm.peersFunc()
	dialed := slices.DeleteFunc(peers, (*p2p.Peer).Inbound)
	return len(peers), len(dialed), len(peers) - len(dialed)
}

// dropRandomPeer selects one of the peers randomly and drops it from the peer pool.
func (cm *dropper) dropRandomPeer(dialed bool) bool {
	peers := cm.peersFunc()

	selectDoNotDrop := func(p *p2p.Peer) bool {
		if dialed {
			// Only drop from dyndialed peers. Avoid dropping trusted peers.
			// Give some time to peers before considering them for a drop.
			return !p.DynDialed() ||
				p.Trusted() ||
				p.Lifetime() < mclock.AbsTime(doNotDropBefore)
		} else {
			// Only drop from inbound peers. Avoid dropping trusted peers.
			// Give some time to peers before considering them for a drop.
			return p.DynDialed() || p.StaticDialed() ||
				p.Trusted() ||
				p.Lifetime() < mclock.AbsTime(doNotDropBefore)
		}
	}
	droppable := slices.DeleteFunc(peers, selectDoNotDrop)
	if len(droppable) > 0 {
		p := droppable[cm.rand.Intn(len(droppable))]
		cm.log.Debug("dropping random peer", "id", p.ID(), "duration", common.PrettyDuration(p.Lifetime()),
			"dialed", dialed, "peercountbefore", len(peers))
		p.Disconnect(p2p.DiscTooManyPeers)
		return true
	}
	return false
}

// randomDuration generates a random duration between min and max.
// TODO: maybe we should move this to a common utlity package.
// TODO: panic might be too harsh, maybe return an error.
func randomDuration(rand *mrand.Rand, min, max time.Duration) time.Duration {
	if min > max {
		panic("min duration must be less than or equal to max duration")
	}
	nanos := rand.Int63n(max.Nanoseconds()-min.Nanoseconds()) + min.Nanoseconds()
	return time.Duration(nanos)
}

// updatePeerDropTimers checks and starts/stops the timer for peer drop.
func (cm *dropper) updatePeerDropTimers(syncing bool) {
	numPeers, numDialed, numInbound := cm.numPeers()
	cm.log.Trace("Dropper status", "syncing", syncing,
		"peers", numPeers, "out", numDialed, "in", numInbound,
		"maxout", cm.maxDialPeers, "maxin", cm.maxInboundPeers)

	if !syncing {
		// If a drop was already scheduled, Schedule does nothing.
		if cm.maxDialPeers-numDialed <= peerDropThreshold {
			interval := randomDuration(cm.rand, peerDropIntervalMin, peerDropIntervalMax)
			cm.peerDropDialedTimer.Schedule(cm.clock.Now().Add(interval))
		} else {
			cm.peerDropDialedTimer.Stop()
		}

		if cm.maxInboundPeers-numInbound <= peerDropThreshold {
			interval := randomDuration(cm.rand, peerDropIntervalMin, peerDropIntervalMax)
			cm.peerDropInboundTimer.Schedule(cm.clock.Now().Add(interval))
		} else {
			cm.peerDropInboundTimer.Stop()
		}
	} else {
		// Downloader is managing connections while syncing.
		cm.peerDropDialedTimer.Stop()
		cm.peerDropInboundTimer.Stop()
	}
}

// loop is the main loop of the connection dropper.
func (cm *dropper) loop() {
	defer cm.wg.Done()

	// Set up periodic timer to pull syncing status.
	// We could get syncing status in a few ways:
	// - poll the sync status (we use this for now)
	// - subscribe to Downloader.mux
	// - subscribe to DownloaderAPI (which itself polls the sync status)
	syncing := cm.syncingFunc()
	cm.log.Trace("Sync status", "syncing", syncing)
	syncCheckTimer := mclock.NewAlarm(cm.dropperConfig.clock)
	syncCheckTimer.Schedule(cm.clock.Now().Add(syncCheckInterval))
	defer syncCheckTimer.Stop()

	for {
		select {
		case <-syncCheckTimer.C():
			// Update info about syncing status, and rearm the timers.
			syncingNew := cm.syncingFunc()
			if syncing != syncingNew {
				// Syncing status changed, we might need to update the timers.
				cm.log.Trace("Sync status changed", "syncing", syncingNew)
				syncing = syncingNew
				cm.updatePeerDropTimers(syncing)
			}
			syncCheckTimer.Schedule(cm.clock.Now().Add(syncCheckInterval))
		case ev := <-cm.peerEventCh:
			if ev.Type == p2p.PeerEventTypeAdd || ev.Type == p2p.PeerEventTypeDrop {
				// Number of peers changed, we might need to start the timers.
				cm.updatePeerDropTimers(syncing)
			}
		case <-cm.peerDropDialedTimer.C():
			cm.dropRandomPeer(true)
		case <-cm.peerDropInboundTimer.C():
			cm.dropRandomPeer(false)
		case <-cm.shutdownCh:
			return
		}
	}
}
