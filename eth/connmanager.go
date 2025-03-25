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
	// Interval between peer drop events
	peerDropInterval = 5 * time.Minute
	// Avoid dropping peers for some time after connection
	doNotDropBefore = 2 * peerDropInterval
	// How close to max should we initiate the drop timer. O should be fine,
	// dropping when no more peers can be added. Larger numbers result in more
	// aggressive drop behavior.
	peerDropThreshold = 0
)

// connManager monitors the state of the peer pool and makes changes as follows:
//   - if the peer count is close to the limit, it drops peers randomly every
//     peerDropInterval to make space for new peers
type connManager struct {
	connmanConfig
	peersFunc getPeersFunc
	syncFunc  getSyncFunc

	// the peerDrop timer introduces churn if we are close to limit capacity
	peerDropTimer *mclock.Alarm
	peerEventCh   chan *p2p.PeerEvent
	sub           event.Subscription

	wg         sync.WaitGroup
	shutdownCh chan struct{}
}

// callback type to get the list of connected peers.
type getPeersFunc func() []*p2p.Peer

// callback type to get sync status.
type getSyncFunc func() bool

type connmanConfig struct {
	maxDialPeers int // maximum number of dialed peers
	log          log.Logger
	clock        mclock.Clock
	rand         *mrand.Rand
}

func (cfg connmanConfig) withDefaults() connmanConfig {
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

func newConnManager(config *connmanConfig) *connManager {
	cfg := config.withDefaults()
	cm := &connManager{
		connmanConfig: cfg,
		peerDropTimer: mclock.NewAlarm(cfg.clock),
		peerEventCh:   make(chan *p2p.PeerEvent),
		shutdownCh:    make(chan struct{}),
	}
	cm.log.Info("New Connection Manager", "maxDialPeers", cm.maxDialPeers, "threshold", peerDropThreshold, "interval", peerDropInterval)
	return cm
}

func (cm *connManager) Start(srv *p2p.Server, syncFunc getSyncFunc) {
	cm.wg.Add(1)
	cm.peersFunc = srv.Peers
	cm.syncFunc = syncFunc
	cm.sub = srv.SubscribeEvents(cm.peerEventCh)
	go cm.loop()
}

// stop the connection manager.
func (cm *connManager) Stop() {
	cm.sub.Unsubscribe()
	cm.peerDropTimer.Stop()
	close(cm.shutdownCh)
	cm.wg.Wait()
}

// numDialPeers returns the current number of peers dialed (not inbound).
func (cm *connManager) numDialPeers() int {
	dialed := slices.DeleteFunc(cm.peersFunc(), (*p2p.Peer).Inbound)
	return len(dialed)
}

func (cm *connManager) numPeers() (int, int, int) {
	peers := cm.peersFunc()
	dialed := slices.DeleteFunc(peers, (*p2p.Peer).Inbound)
	return len(peers), len(dialed), len(peers) - len(dialed)
}

// dropRandomPeer selects one of the peers randomly and drops it from the peer pool.
func (cm *connManager) dropRandomPeer() bool {
	peers := cm.peersFunc()

	// Only drop from dyndialed peers. Avoid dropping trusted peers.
	// Give some time to peers before considering them for a drop.
	selectDoNotDrop := func(p *p2p.Peer) bool {
		return !p.DynDialed() ||
			p.Trusted() ||
			p.Lifetime() < mclock.AbsTime(doNotDropBefore)
	}
	droppable := slices.DeleteFunc(peers, selectDoNotDrop)
	if len(droppable) > 0 {
		p := droppable[cm.rand.Intn(len(droppable))]
		cm.log.Debug("dropping random peer", "id", p.ID(), "duration", common.PrettyDuration(p.Lifetime()), "peercountbefore", len(peers))
		p.Disconnect(p2p.DiscTooManyPeers)
		return true
	}
	return false
}

// loop is the main loop of the connection manager.
func (cm *connManager) loop() {
	defer cm.wg.Done()

	for {
		select {
		case ev := <-cm.peerEventCh:
			switch ev.Type {
			case p2p.PeerEventTypeAdd:
				// check and start timer for peer drop
				// If a drop was already scheduled, Schedule does nothing.
				numpeers, out, in := cm.numPeers()
				cm.log.Trace("addPeerCh", "peers", numpeers, "out", out, "in", in, "maxout", cm.maxDialPeers)
				if cm.maxDialPeers-cm.numDialPeers() <= peerDropThreshold {
					cm.peerDropTimer.Schedule(cm.clock.Now().Add(peerDropInterval))
				}

			case p2p.PeerEventTypeDrop:
				// check and stop timer for peer drop
				numpeers, out, in := cm.numPeers()
				cm.log.Trace("remPeerCh", "peers", numpeers, "out", out, "in", in, "maxout", cm.maxDialPeers)
				if cm.maxDialPeers-cm.numDialPeers() > peerDropThreshold {
					cm.peerDropTimer.Stop()
				}
			}

		case <-cm.peerDropTimer.C():
			cm.dropRandomPeer()
		case <-cm.shutdownCh:
			return
		}
	}
}
