// Copyright 2019 The go-ethereum Authors
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

package les

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/ethdb"
	lps "github.com/ethereum/go-ethereum/les/lespay/server"
	"github.com/ethereum/go-ethereum/les/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/p2p/nodestate"
)

const (
	defaultNegExpTC = 3600 // default time constant (in seconds) for exponentially reducing negative balance

	// defaultConnectedBias is applied to already connected clients So that
	// already connected client won't be kicked out very soon and we
	// can ensure all connected clients can have enough time to request
	// or sync some data.
	//
	// todo(rjl493456442) make it configurable. It can be the option of
	// free trial time!
	defaultConnectedBias = time.Minute * 3
	inactiveTimeout      = time.Second * 10
)

// clientPool implements a client database that assigns a priority to each client
// based on a positive and negative balance. Positive balance is externally assigned
// to prioritized clients and is decreased with connection time and processed
// requests (unless the price factors are zero). If the positive balance is zero
// then negative balance is accumulated.
//
// Balance tracking and priority calculation for connected clients is done by
// balanceTracker. activeQueue ensures that clients with the lowest positive or
// highest negative balance get evicted when the total capacity allowance is full
// and new clients with a better balance want to connect.
//
// Already connected nodes receive a small bias in their favor in order to avoid
// accepting and instantly kicking out clients. In theory, we try to ensure that
// each client can have several minutes of connection time.
//
// Balances of disconnected clients are stored in nodeDB including positive balance
// and negative banalce. Boeth positive balance and negative balance will decrease
// exponentially. If the balance is low enough, then the record will be dropped.
type clientPool struct {
	lps.BalanceTrackerSetup
	lps.PriorityPoolSetup
	lock       sync.Mutex
	clock      mclock.Clock
	closed     bool
	removePeer func(enode.ID)
	ns         *nodestate.NodeStateMachine
	pp         *lps.PriorityPool
	bt         *lps.BalanceTracker

	defaultPosFactors, defaultNegFactors lps.PriceFactors
	posExpTC, negExpTC                   uint64
	minCap                               uint64 // The minimal capacity value allowed for any client
	connectedBias                        time.Duration
	capLimit                             uint64
}

// clientPoolPeer represents a client peer in the pool.
// Positive balances are assigned to node key while negative balances are assigned
// to freeClientId. Currently network IP address without port is used because
// clients have a limited access to IP addresses while new node keys can be easily
// generated so it would be useless to assign a negative value to them.
type clientPoolPeer interface {
	Node() *enode.Node
	freeClientId() string
	updateCapacity(uint64)
	freeze()
	allowInactive() bool
}

// clientInfo defines all information required by clientpool.
type clientInfo struct {
	node                *enode.Node
	address             string
	peer                clientPoolPeer
	connected, priority bool
	connectedAt         mclock.AbsTime
	balance             *lps.NodeBalance
}

// newClientPool creates a new client pool
func newClientPool(ns *nodestate.NodeStateMachine, lespayDb ethdb.Database, minCap uint64, connectedBias time.Duration, clock mclock.Clock, removePeer func(enode.ID)) *clientPool {
	pool := &clientPool{
		ns:                  ns,
		BalanceTrackerSetup: balanceTrackerSetup,
		PriorityPoolSetup:   priorityPoolSetup,
		clock:               clock,
		minCap:              minCap,
		connectedBias:       connectedBias,
		removePeer:          removePeer,
	}
	pool.bt = lps.NewBalanceTracker(ns, balanceTrackerSetup, lespayDb, clock, &utils.Expirer{}, &utils.Expirer{})
	pool.pp = lps.NewPriorityPool(ns, priorityPoolSetup, clock, minCap, connectedBias, 4)

	// set default expiration constants used by tests
	// Note: server overwrites this if token sale is active
	pool.bt.SetExpirationTCs(0, defaultNegExpTC)

	ns.SubscribeState(pool.InactiveFlag.Or(pool.PriorityFlag), func(node *enode.Node, oldState, newState nodestate.Flags) {
		if newState.Equals(pool.InactiveFlag) {
			ns.AddTimeout(node, pool.InactiveFlag, inactiveTimeout)
		}
		if oldState.Equals(pool.InactiveFlag) && newState.Equals(pool.InactiveFlag.Or(pool.PriorityFlag)) {
			ns.SetStateSub(node, pool.InactiveFlag, nodestate.Flags{}, 0) // remove timeout
		}
	})

	ns.SubscribeState(pool.ActiveFlag.Or(pool.PriorityFlag), func(node *enode.Node, oldState, newState nodestate.Flags) {
		c, _ := ns.GetField(node, clientInfoField).(*clientInfo)
		if c == nil {
			return
		}
		c.priority = newState.HasAll(pool.PriorityFlag)
		if newState.Equals(pool.ActiveFlag) {
			cap, _ := ns.GetField(node, pool.CapacityField).(uint64)
			if cap > minCap {
				pool.pp.RequestCapacity(node, minCap, 0, true)
			}
		}
	})

	ns.SubscribeState(pool.InactiveFlag.Or(pool.ActiveFlag), func(node *enode.Node, oldState, newState nodestate.Flags) {
		if oldState.IsEmpty() {
			clientConnectedMeter.Mark(1)
			log.Debug("Client connected", "id", node.ID())
		}
		if oldState.Equals(pool.InactiveFlag) && newState.Equals(pool.ActiveFlag) {
			clientActivatedMeter.Mark(1)
			log.Debug("Client activated", "id", node.ID())
		}
		if oldState.Equals(pool.ActiveFlag) && newState.Equals(pool.InactiveFlag) {
			clientDeactivatedMeter.Mark(1)
			log.Debug("Client deactivated", "id", node.ID())
			c, _ := ns.GetField(node, clientInfoField).(*clientInfo)
			if c == nil || !c.peer.allowInactive() {
				pool.removePeer(node.ID())
			}
		}
		if newState.IsEmpty() {
			clientDisconnectedMeter.Mark(1)
			log.Debug("Client disconnected", "id", node.ID())
			pool.removePeer(node.ID())
		}
	})

	var totalConnected uint64
	ns.SubscribeField(pool.CapacityField, func(node *enode.Node, state nodestate.Flags, oldValue, newValue interface{}) {
		oldCap, _ := oldValue.(uint64)
		newCap, _ := newValue.(uint64)
		totalConnected += newCap - oldCap
		totalConnectedGauge.Update(int64(totalConnected))
		c, _ := ns.GetField(node, clientInfoField).(*clientInfo)
		if c != nil {
			c.peer.updateCapacity(newCap)
		}
	})
	return pool
}

// stop shuts the client pool down
func (f *clientPool) stop() {
	f.lock.Lock()
	f.closed = true
	f.lock.Unlock()
	f.ns.ForEach(nodestate.Flags{}, nodestate.Flags{}, func(node *enode.Node, state nodestate.Flags) {
		// enforces saving all balances in BalanceTracker
		f.disconnectNode(node)
	})
	f.bt.Stop()
}

// connect should be called after a successful handshake. If the connection was
// rejected, there is no need to call disconnect.
func (f *clientPool) connect(peer clientPoolPeer) (uint64, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	// Short circuit if clientPool is already closed.
	if f.closed {
		return 0, fmt.Errorf("Client pool is already closed")
	}
	// Dedup connected peers.
	node, freeID := peer.Node(), peer.freeClientId()
	if f.ns.GetField(node, clientInfoField) != nil {
		log.Debug("Client already connected", "address", freeID, "id", node.ID().String())
		return 0, fmt.Errorf("Client already connected address=%s id=%s", freeID, node.ID().String())
	}
	now := f.clock.Now()
	c := &clientInfo{
		node:        node,
		address:     freeID,
		peer:        peer,
		connected:   true,
		connectedAt: now,
	}
	f.ns.SetField(node, clientInfoField, c)
	f.ns.SetField(node, connAddressField, freeID)
	if c.balance, _ = f.ns.GetField(node, f.BalanceField).(*lps.NodeBalance); c.balance == nil {
		f.disconnect(peer)
		return 0, nil
	}
	c.balance.SetPriceFactors(f.defaultPosFactors, f.defaultNegFactors)

	f.ns.SetState(node, f.InactiveFlag, nodestate.Flags{}, 0)
	var allowed bool
	f.ns.Operation(func() {
		_, allowed = f.pp.RequestCapacity(node, f.minCap, f.connectedBias, true)
	})
	if allowed {
		return f.minCap, nil
	}
	if !peer.allowInactive() {
		f.disconnect(peer)
	}
	return 0, nil
}

// setConnectedBias sets the connection bias, which is applied to already connected clients
// So that already connected client won't be kicked out very soon and we can ensure all
// connected clients can have enough time to request or sync some data.
func (f *clientPool) setConnectedBias(bias time.Duration) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.connectedBias = bias
	f.pp.SetActiveBias(bias)
}

// disconnect should be called when a connection is terminated. If the disconnection
// was initiated by the pool itself using disconnectFn then calling disconnect is
// not necessary but permitted.
func (f *clientPool) disconnect(p clientPoolPeer) {
	f.disconnectNode(p.Node())
}

// disconnectNode removes node fields and flags related to connected status
func (f *clientPool) disconnectNode(node *enode.Node) {
	f.ns.SetField(node, connAddressField, nil)
	f.ns.SetField(node, clientInfoField, nil)
}

// setDefaultFactors sets the default price factors applied to subsequently connected clients
func (f *clientPool) setDefaultFactors(posFactors, negFactors lps.PriceFactors) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.defaultPosFactors = posFactors
	f.defaultNegFactors = negFactors
}

// capacityInfo returns the total capacity allowance, the total capacity of connected
// clients and the total capacity of connected and prioritized clients
func (f *clientPool) capacityInfo() (uint64, uint64, uint64) {
	f.lock.Lock()
	defer f.lock.Unlock()

	// total priority active cap will be supported when the token issuer module is added
	_, activeCap := f.pp.Active()
	return f.capLimit, activeCap, 0
}

// setLimits sets the maximum number and total capacity of connected clients,
// dropping some of them if necessary.
func (f *clientPool) setLimits(totalConn int, totalCap uint64) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.capLimit = totalCap
	f.pp.SetLimits(uint64(totalConn), totalCap)
}

// setCapacity sets the assigned capacity of a connected client
func (f *clientPool) setCapacity(node *enode.Node, freeID string, capacity uint64, bias time.Duration, setCap bool) (uint64, error) {
	c, _ := f.ns.GetField(node, clientInfoField).(*clientInfo)
	if c == nil {
		if setCap {
			return 0, fmt.Errorf("client %064x is not connected", node.ID())
		}
		c = &clientInfo{node: node}
		f.ns.SetField(node, clientInfoField, c)
		f.ns.SetField(node, connAddressField, freeID)
		if c.balance, _ = f.ns.GetField(node, f.BalanceField).(*lps.NodeBalance); c.balance == nil {
			log.Error("BalanceField is missing", "node", node.ID())
			return 0, fmt.Errorf("BalanceField of %064x is missing", node.ID())
		}
		defer func() {
			f.ns.SetField(node, connAddressField, nil)
			f.ns.SetField(node, clientInfoField, nil)
		}()
	}
	var (
		minPriority int64
		allowed     bool
	)
	f.ns.Operation(func() {
		if !setCap || c.priority {
			// check clientInfo.priority inside Operation to ensure thread safety
			minPriority, allowed = f.pp.RequestCapacity(node, capacity, bias, setCap)
		}
	})
	if allowed {
		return 0, nil
	}
	missing := c.balance.PosBalanceMissing(minPriority, capacity, bias)
	if missing < 1 {
		// ensure that we never return 0 missing and insufficient priority error
		missing = 1
	}
	return missing, errNoPriority
}

// setCapacityLocked is the equivalent of setCapacity used when f.lock is already locked
func (f *clientPool) setCapacityLocked(node *enode.Node, freeID string, capacity uint64, minConnTime time.Duration, setCap bool) (uint64, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	return f.setCapacity(node, freeID, capacity, minConnTime, setCap)
}

// forClients calls the supplied callback for either the listed node IDs or all connected
// nodes. It passes a valid clientInfo to the callback and ensures that the necessary
// fields and flags are set in order for BalanceTracker and PriorityPool to work even if
// the node is not connected.
func (f *clientPool) forClients(ids []enode.ID, cb func(client *clientInfo)) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if len(ids) == 0 {
		f.ns.ForEach(nodestate.Flags{}, nodestate.Flags{}, func(node *enode.Node, state nodestate.Flags) {
			c, _ := f.ns.GetField(node, clientInfoField).(*clientInfo)
			if c != nil {
				cb(c)
			}
		})
	} else {
		for _, id := range ids {
			node := f.ns.GetNode(id)
			if node == nil {
				node = enode.SignNull(&enr.Record{}, id)
			}
			c, _ := f.ns.GetField(node, clientInfoField).(*clientInfo)
			if c != nil {
				cb(c)
			} else {
				c = &clientInfo{node: node}
				f.ns.SetField(node, clientInfoField, c)
				f.ns.SetField(node, connAddressField, "")
				if c.balance, _ = f.ns.GetField(node, f.BalanceField).(*lps.NodeBalance); c.balance != nil {
					cb(c)
				} else {
					log.Error("BalanceField is missing")
				}
				f.ns.SetField(node, connAddressField, nil)
				f.ns.SetField(node, clientInfoField, nil)
			}
		}
	}
}
