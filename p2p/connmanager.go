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

package p2p

import (
	crand "crypto/rand"
	"encoding/binary"
	mrand "math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/log"
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

	// the peerDrop timer introduces churn if we are close to limit capacity
	peerDropTimer *mclock.Alarm
	addPeerCh     chan *conn
	remPeerCh     chan *conn
}

// callback type to get the list of connected peers.
type getPeersFunc func() []*Peer

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

func newConnManager(config connmanConfig, peersFunc getPeersFunc) *connManager {
	cfg := config.withDefaults()
	cm := &connManager{
		connmanConfig: cfg,
		peerDropTimer: mclock.NewAlarm(cfg.clock),
		peersFunc:     peersFunc,
		addPeerCh:     make(chan *conn),
		remPeerCh:     make(chan *conn),
	}
	cm.log.Info("New Connection Manager", "maxDialPeers", cm.maxDialPeers, "threshold", peerDropThreshold, "interval", peerDropInterval)
	go cm.loop()
	return cm
}

// stop the connection manager.
func (cm *connManager) stop() {
	cm.peerDropTimer.Stop()
}

// peerAdded notifies about peerset change.
func (cm *connManager) peerAdded(c *conn) {
	cm.addPeerCh <- c
}

// peerRemoved notifies about peerset change.
func (cm *connManager) peerRemoved(c *conn) {
	cm.remPeerCh <- c
}

// filter is a helper function to filter the peerset.
func filter[T any](s []T, test func(T) bool) (filtered []T) {
	for _, a := range s {
		if test(a) {
			filtered = append(filtered, a)
		}
	}
	return
}

// numDialPeers returns the current number of peers dialed (not inbound).
func (cm *connManager) numDialPeers() int {
	selectDialed := func(p *Peer) bool { return !p.rw.is(inboundConn) }
	dialed := filter(cm.peersFunc(), selectDialed)
	return len(dialed)
}

func (cm *connManager) numPeers() (int, int, int) {
	selectDialed := func(p *Peer) bool { return !p.rw.is(inboundConn) }
	peers := cm.peersFunc()
	dialed := filter(peers, selectDialed)
	return len(peers), len(dialed), len(peers) - len(dialed)
}

// dropRandomPeer selects one of the peers randomly and drops it from the peer pool.
func (cm *connManager) dropRandomPeer() bool {
	peers := cm.peersFunc()

	// Only drop from dyndialed peers. Avoid dropping trusted peers.
	// Give some time to peers before considering them for a drop.
	selectDroppable := func(p *Peer) bool {
		return p.rw.is(dynDialedConn) && !p.rw.is(trustedConn) &&
			mclock.Now()-p.created >= mclock.AbsTime(doNotDropBefore)
	}
	droppable := filter(peers, selectDroppable)
	if len(droppable) > 0 {
		p := droppable[cm.rand.Intn(len(droppable))]
		cm.log.Trace("dropping random peer", "id", p.ID(), "duration", common.PrettyDuration(mclock.Now()-p.created), "peercountbefore", len(peers))
		p.Disconnect(DiscDropped)
		return true
	}
	return false
}

// loop is the main loop of the connection manager.
func (cm *connManager) loop() {

	for {

		select {

		case <-cm.addPeerCh:
			// check and start timer for peer drop
			// If a drop was already scheduled, Schedule does nothing.
			numpeers, out, in := cm.numPeers()
			cm.log.Trace("addPeerCh", "peers", numpeers, "out", out, "in", in, "maxout", cm.maxDialPeers)
			if cm.maxDialPeers-cm.numDialPeers() <= peerDropThreshold {
				cm.peerDropTimer.Schedule(cm.clock.Now().Add(peerDropInterval))
			}

		case <-cm.remPeerCh:
			// check and stop timer for peer drop
			numpeers, out, in := cm.numPeers()
			cm.log.Trace("remPeerCh", "peers", numpeers, "out", out, "in", in, "maxout", cm.maxDialPeers)
			if cm.maxDialPeers-cm.numDialPeers() > peerDropThreshold {
				cm.peerDropTimer.Stop()
			}

		case <-cm.peerDropTimer.C():
			cm.dropRandomPeer()
		}
	}
	cm.log.Warn("Exiting connmanager loop")
}
