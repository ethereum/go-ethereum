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
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
	lru "github.com/hashicorp/golang-lru"
)

const (
	negBalanceExpTC              = time.Hour        // time constant for exponentially reducing negative balance
	fixedPointMultiplier         = 0x1000000        // constant to convert logarithms to fixed point format
	lazyQueueRefresh             = time.Second * 10 // refresh period of the connected queue
	tryActivatePeriod            = time.Second * 5  // periodically check whether inactive clients can be activated
	dropInactiveCycles           = 2                // number of activation check periods after non-priority inactive peers are dropped
	persistCumulativeTimeRefresh = time.Minute * 5  // refresh period of the cumulative running time persistence
	posBalanceCacheLimit         = 8192             // the maximum number of cached items in positive balance queue
	negBalanceCacheLimit         = 8192             // the maximum number of cached items in negative balance queue
	fullRatioTC                  = time.Hour

	// activeBias is applied to already connected clients So that
	// already connected client won't be kicked out very soon and we
	// can ensure all connected clients can have enough time to request
	// or sync some data.
	//
	// todo(rjl493456442) make it configurable. It can be the option of
	// free trial time!
	activeBias = time.Minute * 3
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
// and negative banalce. Negative balance is transformed into a logarithmic form
// with a constantly shifting linear offset in order to implement an exponential
// decrease. Besides nodeDB will have a background thread to check the negative
// balance of disconnected client. If the balance is low enough, then the record
// will be dropped.
type clientPool struct {
	ndb        *nodeDB
	lock       sync.Mutex
	clock      mclock.Clock
	stopCh     chan struct{}
	closed     bool
	removePeer func(enode.ID)

	connectedMap        map[enode.ID]*clientInfo
	activeQueue         *prque.LazyQueue
	inactiveQueue       *prque.Prque
	dropInactivePeers   map[uint64][]*clientInfo
	dropInactiveCounter uint64

	activeBalances, inactiveBalances                uint64
	lastConnectedBalanceUpdate, fullRatioLastUpdate mclock.AbsTime
	fullRatio                                       float64

	defaultPosFactors, defaultNegFactors priceFactors

	activeLimit    int            // The maximum number of connections that clientpool can support
	capLimit       uint64         // The maximum cumulative capacity that clientpool can support
	activeCap      uint64         // The sum of the capacity of the current clientpool connected
	priorityActive uint64         // The sum of the capacity of currently connected priority clients
	minCap         uint64         // The minimal capacity value allowed for any client
	freeClientCap  uint64         // The capacity value of each free client
	startTime      mclock.AbsTime // The timestamp at which the clientpool started running
	cumulativeTime int64          // The cumulative running time of clientpool at the start point.
	disableBias    bool           // Disable connection bias(used in testing)
}

// clientPeer represents a client in the pool.
// Positive balances are assigned to node key while negative balances are assigned
// to freeClientId. Currently network IP address without port is used because
// clients have a limited access to IP addresses while new node keys can be easily
// generated so it would be useless to assign a negative value to them.
type clientPeer interface {
	ID() enode.ID
	freeClientId() string
	updateCapacity(uint64)
	freezeClient()
}

// clientInfo represents a connected client
type clientInfo struct {
	address                string
	id                     enode.ID
	freeID                 string
	active                 bool
	connectedAt            mclock.AbsTime
	capacity               uint64
	priority               bool
	pool                   *clientPool
	peer                   clientPeer
	queueIndex             int // position in activeQueue
	balanceTracker         balanceTracker
	posFactors, negFactors priceFactors
	balanceMetaInfo        string
}

// connSetIndex callback updates clientInfo item index in activeQueue
func connSetIndex(a interface{}, index int) {
	a.(*clientInfo).queueIndex = index
}

// connPriority callback returns actual priority of clientInfo item in activeQueue
func connPriority(a interface{}, now mclock.AbsTime) int64 {
	c := a.(*clientInfo)
	return c.balanceTracker.getPriority(now)
}

// connMaxPriority callback returns estimated maximum priority of clientInfo item in activeQueue
func connMaxPriority(a interface{}, until mclock.AbsTime) int64 {
	c := a.(*clientInfo)
	pri := c.balanceTracker.estimatedPriority(until, true)
	c.balanceTracker.addCallback(balanceCallbackQueue, pri+1, func() {
		c.pool.lock.Lock()
		if c.active && c.queueIndex != -1 {
			c.pool.activeQueue.Update(c.queueIndex)
		}
		c.pool.lock.Unlock()
	})
	return pri
}

// priceFactors determine the pricing policy (may apply either to positive or
// negative balances which may have different factors).
// - timeFactor is cost unit per nanosecond of connection time
// - capacityFactor is cost unit per nanosecond of connection time per 1000000 capacity
// - requestFactor is cost unit per request "realCost" unit
type priceFactors struct {
	timeFactor, capacityFactor, requestFactor float64
}

// newClientPool creates a new client pool
func newClientPool(db ethdb.Database, minCap, freeClientCap uint64, clock mclock.Clock, removePeer func(enode.ID)) *clientPool {
	ndb := newNodeDB(db, clock)
	pool := &clientPool{
		ndb:               ndb,
		clock:             clock,
		connectedMap:      make(map[enode.ID]*clientInfo),
		activeQueue:       prque.NewLazyQueue(connSetIndex, connPriority, connMaxPriority, clock, lazyQueueRefresh),
		inactiveQueue:     prque.New(connSetIndex),
		dropInactivePeers: make(map[uint64][]*clientInfo),
		minCap:            minCap,
		freeClientCap:     freeClientCap,
		removePeer:        removePeer,
		startTime:         clock.Now(),
		cumulativeTime:    ndb.getCumulativeTime(),
		stopCh:            make(chan struct{}),
	}
	// calculate total token balance amount
	var start enode.ID
	for {
		ids := pool.ndb.getPosBalanceIDs(start, enode.ID{}, 1000)
		var stop bool
		l := len(ids)
		if l == 1000 {
			l--
			start = ids[l]
		} else {
			stop = true
		}
		for i := 0; i < l; i++ {
			pool.inactiveBalances += pool.ndb.getOrNewPB(ids[i]).value
		}
		if stop {
			break
		}
	}
	// If the negative balance of free client is even lower than 1,
	// delete this entry.
	ndb.nbEvictCallBack = func(now mclock.AbsTime, b negBalance) bool {
		balance := math.Exp(float64(b.logValue-pool.logOffset(now)) / fixedPointMultiplier)
		return balance <= 1
	}
	go func() {
		for {
			select {
			case <-clock.After(lazyQueueRefresh):
				pool.lock.Lock()
				pool.activeQueue.Refresh()
				pool.lock.Unlock()
			case <-pool.stopCh:
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case <-clock.After(persistCumulativeTimeRefresh):
				pool.ndb.setCumulativeTime(pool.logOffset(clock.Now()))
			case <-pool.stopCh:
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case <-clock.After(tryActivatePeriod):
				pool.lock.Lock()
				pool.tryActivateClients()
				for _, c := range pool.dropInactivePeers[pool.dropInactiveCounter] {
					if _, ok := pool.connectedMap[c.id]; ok && !c.active && !c.priority {
						pool.disconnectLocked(c.peer)
					}
				}
				delete(pool.dropInactivePeers, pool.dropInactiveCounter)
				pool.dropInactiveCounter++
				pool.lock.Unlock()
			case <-pool.stopCh:
				return
			}
		}
	}()
	return pool
}

// stop shuts the client pool down
func (f *clientPool) stop() {
	close(f.stopCh)
	f.lock.Lock()
	f.closed = true
	f.lock.Unlock()
	f.ndb.setCumulativeTime(f.logOffset(f.clock.Now()))
	f.ndb.close()
}

func (f *clientPool) updateFullRatio() {
	full := float64(1)
	if f.priorityActive < f.capLimit {
		freeCap := f.capLimit - f.priorityActive
		if freeCap > f.freeClientCap {
			freeCapThreshold := f.capLimit / 4
			if freeCap > freeCapThreshold {
				full = 0
			} else {
				full = float64(freeCapThreshold-freeCap) / float64(freeCapThreshold-f.freeClientCap)
			}
		}
	}
	now := f.clock.Now()
	dt := now - f.fullRatioLastUpdate
	f.fullRatioLastUpdate = now
	if dt < 0 {
		dt = 0
	}
	d := math.Exp(-float64(dt) / float64(fullRatioTC))
	f.fullRatio = full - (full-f.fullRatio)*d
}

func (f *clientPool) totalTokenLimit() uint64 {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.updateFullRatio()
	d := 1 - f.fullRatio
	if d > 0.5 {
		d = -math.Log(0.5/d) * float64(fullRatioTC)
	} else {
		d = 0
	}
	return uint64(d * float64(f.capLimit) * f.defaultPosFactors.capacityFactor)
}

func (f *clientPool) totalTokenAmount() uint64 {
	f.lock.Lock()
	defer f.lock.Unlock()

	now := f.clock.Now()
	if now > f.lastConnectedBalanceUpdate+mclock.AbsTime(time.Second) {
		f.activeBalances = 0
		for _, c := range f.connectedMap {
			pos, _ := c.balanceTracker.getBalance(now)
			f.activeBalances += pos
		}
		f.lastConnectedBalanceUpdate = now
	}
	return f.activeBalances + f.inactiveBalances
}

// connect should be called after a successful handshake. If the connection was
// rejected, there is no need to call disconnect.
func (f *clientPool) connect(peer clientPeer, reqCapacity uint64) (uint64, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	// Short circuit if clientPool is already closed.
	if f.closed {
		return 0, fmt.Errorf("Client pool is already closed")
	}
	// Dedup connected peers.
	id, freeID := peer.ID(), peer.freeClientId()
	if _, ok := f.connectedMap[id]; ok {
		clientRejectedMeter.Mark(1)
		log.Debug("Client already connected", "address", freeID, "id", peerIdToString(id))
		return 0, fmt.Errorf("Client already connected   address = %s  id = %s", freeID, peerIdToString(id))
	}
	pb := f.ndb.getOrNewPB(id)
	nb := f.ndb.getOrNewNB(freeID)
	e := &clientInfo{
		capacity:        reqCapacity,
		pool:            f,
		peer:            peer,
		address:         freeID,
		queueIndex:      -1,
		id:              id,
		freeID:          freeID,
		connectedAt:     f.clock.Now(),
		priority:        pb.value != 0,
		posFactors:      f.defaultPosFactors,
		negFactors:      f.defaultNegFactors,
		balanceMetaInfo: pb.meta,
	}
	missing, capacity := f.capAvailable(id, freeID, reqCapacity, 0, true)
	f.connectedMap[id] = e
	if missing != 0 {
		// capacity is not available, add client to inactive queue
		f.initBalanceTracker(&e.balanceTracker, pb, nb, capacity, false)
		f.inactiveQueue.Push(e, -connPriority(e, f.clock.Now()))
		return 0, nil
	}
	// capacity is available, add client
	e.active = true
	e.capacity = capacity
	f.initBalanceTracker(&e.balanceTracker, pb, nb, capacity, true)
	// Register new client to connection queue.
	f.inactiveBalances -= pb.value
	f.activeBalances += pb.value
	f.activeQueue.Push(e)
	f.activeCap += e.capacity

	// If the current client is a paid client, monitor the status of client,
	// downgrade it to normal client if positive balance is used up.
	if e.priority {
		f.updateFullRatio()
		f.priorityActive += capacity
		e.balanceTracker.addCallback(balanceCallbackZero, 0, func() { f.balanceExhausted(id) })
	}
	totalConnectedGauge.Update(int64(f.activeCap))
	clientConnectedMeter.Mark(1)
	log.Debug("Client accepted", "address", freeID)
	return e.capacity, nil
}

func (f *clientPool) initBalanceTracker(bt *balanceTracker, pb posBalance, nb negBalance, capacity uint64, active bool) {
	posBalance := pb.value
	var negBalance uint64
	if nb.logValue != 0 {
		negBalance = uint64(math.Exp(float64(nb.logValue-f.logOffset(f.clock.Now()))/fixedPointMultiplier) * float64(time.Second))
	}
	bt.init(f.clock, capacity)
	bt.setBalance(posBalance, negBalance)
	if active {
		updatePriceFactors(bt, f.defaultPosFactors, f.defaultNegFactors, capacity)
	} else {
		zeroPriceFactors(bt)
	}
}

// disconnect should be called when a connection is terminated. If the disconnection
// was initiated by the pool itself using disconnectFn then calling disconnect is
// not necessary but permitted.
func (f *clientPool) disconnect(p clientPeer) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.disconnectLocked(p)
}

func (f *clientPool) disconnectLocked(p clientPeer) {
	// Short circuit if client pool is already closed.
	if f.closed {
		return
	}
	e, ok := f.connectedMap[p.ID()]
	if !ok {
		log.Debug("Client not connected", "address", p.freeClientId(), "id", peerIdToString(p.ID()))
		return
	}
	tryActivate := e.active
	if e.active {
		f.deactivateClient(e, false)
	}
	f.finalizeBalance(e, f.clock.Now())
	f.inactiveQueue.Remove(e.queueIndex)
	delete(f.connectedMap, e.id)
	clientDisconnectedMeter.Mark(1)
	log.Debug("Client disconnected", "address", e.address)
	if tryActivate {
		f.tryActivateClients()
	}
}

// capAvailable checks whether the current priority level of the given client is enough to
// connect or change capacity to the requested level and then stay connected for at least
// the specified duration. If not then the additional required amount of positive balance is returned.
func (f *clientPool) capAvailable(id enode.ID, freeID string, capacity uint64, minConnTime time.Duration, kick bool) (uint64, uint64) {
	var missing uint64
	if capacity == 0 {
		capacity = f.freeClientCap
	}
	if capacity < f.minCap {
		capacity = f.minCap
	}
	newCapacity := f.activeCap + capacity
	newCount := f.activeQueue.Size() + 1
	client := f.connectedMap[id]
	if client != nil && client.active {
		newCapacity -= client.capacity
		newCount--
	}
	if newCapacity > f.capLimit || newCount > f.activeLimit {
		var (
			popList        []*clientInfo
			targetPriority int64
		)
		f.activeQueue.MultiPop(func(data interface{}, priority int64) bool {
			c := data.(*clientInfo)
			popList = append(popList, c)
			if c != client {
				targetPriority = priority
				newCapacity -= c.capacity
				newCount--
			}
			return newCapacity > f.capLimit || newCount > f.activeLimit
		})
		if newCapacity > f.capLimit || newCount > f.activeLimit {
			missing = math.MaxUint64
		} else {
			var bt *balanceTracker
			if client != nil {
				bt = &client.balanceTracker
			} else {
				bt = &balanceTracker{}
				f.initBalanceTracker(bt, f.ndb.getOrNewPB(id), f.ndb.getOrNewNB(freeID), capacity, true)
			}
			if capacity != f.freeClientCap && targetPriority >= 0 {
				targetPriority = -1
			}
			bias := activeBias
			if f.disableBias {
				bias = 0
			}
			if bias < minConnTime {
				bias = minConnTime
			}
			missing = bt.posBalanceMissing(targetPriority, capacity, bias)
		}
		if missing != 0 {
			kick = false
		}
		for _, c := range popList {
			if kick && c != client {
				f.deactivateClient(c, true)
			} else {
				f.activeQueue.Push(c)
			}
		}
	}
	return missing, capacity
}

// forClients iterates through a list of clients, calling the callback for each one.
// If a client is not connected then clientInfo is nil. If the specified list is empty
// then the callback is called for all connected clients.
func (f *clientPool) forClients(ids []enode.ID, callback func(*clientInfo, enode.ID) error) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	if len(ids) > 0 {
		for _, id := range ids {
			if err := callback(f.connectedMap[id], id); err != nil {
				return err
			}
		}
	} else {
		for _, c := range f.connectedMap {
			if err := callback(c, c.id); err != nil {
				return err
			}
		}
	}
	return nil
}

// setDefaultFactors sets the default price factors applied to subsequently connected clients
func (f *clientPool) setDefaultFactors(posFactors, negFactors priceFactors) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.defaultPosFactors = posFactors
	f.defaultNegFactors = negFactors
}

func (f *clientPool) deactivateClient(e *clientInfo, scheduleDrop bool) {
	if _, ok := f.connectedMap[e.id]; !ok || !e.active {
		return
	}
	f.activeQueue.Remove(e.queueIndex)
	f.activeCap -= e.capacity
	if e.priority {
		f.updateFullRatio()
		f.priorityActive -= e.capacity
	}
	e.active = false
	e.peer.updateCapacity(0)
	totalConnectedGauge.Update(int64(f.activeCap))
	f.inactiveQueue.Push(e, -connPriority(e, f.clock.Now()))
	if scheduleDrop {
		f.dropInactivePeers[f.dropInactiveCounter+dropInactiveCycles] = append(f.dropInactivePeers[f.dropInactiveCounter+dropInactiveCycles], e)
	}
}

func (f *clientPool) tryActivateClients() {
	now := f.clock.Now()
	for f.inactiveQueue.Size() != 0 {
		e := f.inactiveQueue.PopItem().(*clientInfo)
		missing, capacity := f.capAvailable(e.id, e.freeID, e.capacity, 0, true)
		if missing != 0 {
			f.inactiveQueue.Push(e, -connPriority(e, now))
			return
		}
		// capacity is available, activate client
		e.active = true
		e.capacity = capacity
		e.peer.updateCapacity(capacity)
		balance, _ := e.balanceTracker.getBalance(now)
		e.balanceTracker.setCapacity(capacity)
		updatePriceFactors(&e.balanceTracker, f.defaultPosFactors, f.defaultNegFactors, capacity)
		// Register activated client to connection queue.
		f.inactiveBalances -= balance
		f.activeBalances += balance
		f.activeQueue.Push(e)
		f.activeCap += e.capacity

		// If the current client is a paid client, monitor the status of client,
		// downgrade it to normal client if positive balance is used up.
		if e.priority {
			f.updateFullRatio()
			f.priorityActive += capacity
			e.balanceTracker.addCallback(balanceCallbackZero, 0, func() { f.balanceExhausted(e.id) })
		}
		e.peer.updateCapacity(e.capacity)
		totalConnectedGauge.Update(int64(f.activeCap))
		clientConnectedMeter.Mark(1)
		log.Debug("Client activated", "address", e.freeID)
	}
}

// capacityInfo returns the total capacity allowance, the total capacity of connected
// clients and the total capacity of connected and prioritized clients
func (f *clientPool) capacityInfo() (uint64, uint64, uint64) {
	f.lock.Lock()
	defer f.lock.Unlock()

	return f.capLimit, f.activeCap, f.priorityActive
}

// finalizeBalance stops the balance tracker, retrieves the final balances and
// stores them in posBalanceQueue and negBalanceQueue
func (f *clientPool) finalizeBalance(c *clientInfo, now mclock.AbsTime) {
	c.balanceTracker.stop(now)
	pos, neg := c.balanceTracker.getBalance(now)
	f.inactiveBalances += pos
	f.activeBalances -= pos

	pb, nb := f.ndb.getOrNewPB(c.id), f.ndb.getOrNewNB(c.address)
	pb.value = pos
	f.ndb.setPB(c.id, pb)

	neg /= uint64(time.Second) // Convert the expanse to second level.
	if neg > 1 {
		nb.logValue = int64(math.Log(float64(neg))*fixedPointMultiplier) + f.logOffset(now)
		f.ndb.setNB(c.address, nb)
	} else {
		f.ndb.delNB(c.address) // Negative balance is small enough, drop it directly.
	}
}

// balanceExhausted callback is called by balanceTracker when positive balance is exhausted.
// It revokes priority status and also reduces the client capacity if necessary.
func (f *clientPool) balanceExhausted(id enode.ID) {
	f.lock.Lock()
	defer f.lock.Unlock()

	c := f.connectedMap[id]
	if c == nil || !c.priority {
		return
	}
	if c.priority {
		f.updateFullRatio()
		f.priorityActive -= c.capacity
	}
	c.priority = false
	if c.capacity != f.freeClientCap {
		f.activeCap += f.freeClientCap - c.capacity
		totalConnectedGauge.Update(int64(f.activeCap))
		c.capacity = f.freeClientCap
		c.balanceTracker.setCapacity(c.capacity)
		c.peer.updateCapacity(c.capacity)
	}
	pb := f.ndb.getOrNewPB(id)
	pb.value = 0
	f.ndb.setPB(id, pb)
}

// setactiveLimit sets the maximum number and total capacity of connected clients,
// dropping some of them if necessary.
func (f *clientPool) setLimits(totalConn int, totalCap uint64) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.updateFullRatio()
	f.activeLimit = totalConn
	f.capLimit = totalCap
	if f.activeCap > f.capLimit || f.activeQueue.Size() > f.activeLimit {
		f.activeQueue.MultiPop(func(data interface{}, priority int64) bool {
			f.deactivateClient(data.(*clientInfo), true)
			return f.activeCap > f.capLimit || f.activeQueue.Size() > f.activeLimit
		})
	} else {
		f.tryActivateClients()
	}
}

// setCapacity sets the assigned capacity of a connected client
func (f *clientPool) setCapacity(id enode.ID, freeID string, capacity uint64, minConnTime time.Duration, setCap bool) (uint64, uint64, error) {
	c := f.connectedMap[id]
	if c != nil {
		if c.capacity == capacity {
			return 0, capacity, nil
		}
	}
	var missing uint64
	missing, capacity = f.capAvailable(id, freeID, capacity, 0, setCap && c != nil)
	if missing != 0 {
		return missing, capacity, errNoPriority
	}
	// capacity update is possible
	if setCap {
		if c == nil {
			return 0, capacity, fmt.Errorf("client %064x is not connected", c.id[:])
		}
		f.activeCap += capacity - c.capacity
		f.updateFullRatio()
		f.priorityActive += capacity - c.capacity
		c.capacity = capacity
		c.balanceTracker.setCapacity(capacity)
		f.activeQueue.Update(c.queueIndex)
		totalConnectedGauge.Update(int64(f.activeCap))
		updatePriceFactors(&c.balanceTracker, c.posFactors, c.negFactors, c.capacity)
		c.peer.updateCapacity(c.capacity)
		f.tryActivateClients()
	}
	return 0, capacity, nil
}

func (f *clientPool) setCapacityLocked(id enode.ID, freeID string, capacity uint64, minConnTime time.Duration, setCap bool) (uint64, uint64, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	return f.setCapacity(id, freeID, capacity, minConnTime, setCap)
}

// requestCost feeds request cost after serving a request from the given peer and
// returns the remaining token balance
func (f *clientPool) requestCost(p *peer, cost uint64) uint64 {
	f.lock.Lock()
	defer f.lock.Unlock()

	c := f.connectedMap[p.ID()]
	if c == nil || f.closed {
		return 0
	}
	return c.balanceTracker.requestCost(cost)
}

// logOffset calculates the time-dependent offset for the logarithmic
// representation of negative balance
//
// From another point of view, the result returned by the function represents
// the total time that the clientpool is cumulatively running(total_hours/multiplier).
func (f *clientPool) logOffset(now mclock.AbsTime) int64 {
	// Note: fixedPointMultiplier acts as a multiplier here; the reason for dividing the divisor
	// is to avoid int64 overflow. We assume that int64(negBalanceExpTC) >> fixedPointMultiplier.
	cumulativeTime := int64((time.Duration(now - f.startTime)) / (negBalanceExpTC / fixedPointMultiplier))
	return f.cumulativeTime + cumulativeTime
}

// updatePriceFactors sets the pricing factors for an individual connected client
func updatePriceFactors(bt *balanceTracker, posFactors, negFactors priceFactors, capacity uint64) {
	bt.setFactors(true, negFactors.timeFactor+float64(capacity)*negFactors.capacityFactor/1000000, negFactors.requestFactor)
	bt.setFactors(false, posFactors.timeFactor+float64(capacity)*posFactors.capacityFactor/1000000, posFactors.requestFactor)
}

func zeroPriceFactors(bt *balanceTracker) {
	bt.setFactors(true, 0, 0)
	bt.setFactors(false, 0, 0)
}

// getPosBalance retrieves a single positive balance entry from cache or the database
func (f *clientPool) getPosBalance(id enode.ID) posBalance {
	f.lock.Lock()
	defer f.lock.Unlock()

	if c := f.connectedMap[id]; c != nil {
		pb, _ := c.balanceTracker.getBalance(mclock.Now())
		return posBalance{value: pb, meta: c.balanceMetaInfo}
	} else {
		return f.ndb.getOrNewPB(id)
	}
}

// addBalance updates the balance of a client (either overwrites it or adds to it).
// It also updates the balance meta info string.
func (f *clientPool) addBalance(id enode.ID, amount int64, meta string) (uint64, uint64, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	pb := f.ndb.getOrNewPB(id)
	var negBalance uint64
	c := f.connectedMap[id]
	if c != nil {
		pb.value, negBalance = c.balanceTracker.getBalance(f.clock.Now())
	}
	oldBalance := pb.value
	if amount > 0 {
		if amount > maxBalance || pb.value > maxBalance-uint64(amount) {
			return oldBalance, oldBalance, errBalanceOverflow
		}
		pb.value += uint64(amount)
	} else {
		if uint64(-amount) > pb.value {
			pb.value = 0
		} else {
			pb.value -= uint64(-amount)
		}
	}
	pb.meta = meta
	f.ndb.setPB(id, pb)
	if c != nil {
		c.balanceTracker.setBalance(pb.value, negBalance)
		if c.active {
			f.activeQueue.Update(c.queueIndex)
			if !c.priority && pb.value > 0 {
				// The capacity should be adjusted based on the requirement,
				// but we have no idea about the new capacity, need a second
				// call to udpate it.
				f.updateFullRatio()
				f.priorityActive += c.capacity
				c.balanceTracker.addCallback(balanceCallbackZero, 0, func() { f.balanceExhausted(id) })
			}
			c.balanceMetaInfo = meta
			f.activeBalances += pb.value - oldBalance
		} else {
			f.inactiveQueue.Remove(c.queueIndex)
			f.inactiveQueue.Push(c, -connPriority(c, f.clock.Now()))
			f.inactiveBalances += pb.value - oldBalance
		}
		if pb.value > 0 {
			c.priority = true
			// if balance is set to zero then reverting to non-priority status
			// is handled by the balanceExhausted callback
		}
	} else {
		f.inactiveBalances += pb.value - oldBalance
	}
	f.tryActivateClients()
	return oldBalance, pb.value, nil
}

// posBalance represents a recently accessed positive balance entry
type posBalance struct {
	value uint64
	meta  string
}

// EncodeRLP implements rlp.Encoder
func (e *posBalance) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{e.value, e.meta})
}

// DecodeRLP implements rlp.Decoder
func (e *posBalance) DecodeRLP(s *rlp.Stream) error {
	var entry struct {
		Value uint64
		Meta  string
	}
	if err := s.Decode(&entry); err != nil {
		return err
	}
	e.value = entry.Value
	e.meta = entry.Meta
	return nil
}

// negBalance represents a negative balance entry of a disconnected client
type negBalance struct{ logValue int64 }

// EncodeRLP implements rlp.Encoder
func (e *negBalance) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{uint64(e.logValue)})
}

// DecodeRLP implements rlp.Decoder
func (e *negBalance) DecodeRLP(s *rlp.Stream) error {
	var entry struct {
		LogValue uint64
	}
	if err := s.Decode(&entry); err != nil {
		return err
	}
	e.logValue = int64(entry.LogValue)
	return nil
}

const (
	// nodeDBVersion is the version identifier of the node data in db
	//
	// Changelog:
	// * Replace `lastTotal` with `meta` in positive balance: version 0=>1
	nodeDBVersion = 1

	// dbCleanupCycle is the cycle of db for useless data cleanup
	dbCleanupCycle = time.Hour
)

var (
	positiveBalancePrefix    = []byte("pb:")             // dbVersion(uint16 big endian) + positiveBalancePrefix + id -> balance
	negativeBalancePrefix    = []byte("nb:")             // dbVersion(uint16 big endian) + negativeBalancePrefix + ip -> balance
	cumulativeRunningTimeKey = []byte("cumulativeTime:") // dbVersion(uint16 big endian) + cumulativeRunningTimeKey -> cumulativeTime
)

type nodeDB struct {
	db              ethdb.Database
	pcache          *lru.Cache
	ncache          *lru.Cache
	auxbuf          []byte                                // 37-byte auxiliary buffer for key encoding
	verbuf          [2]byte                               // 2-byte auxiliary buffer for db version
	nbEvictCallBack func(mclock.AbsTime, negBalance) bool // Callback to determine whether the negative balance can be evicted.
	clock           mclock.Clock
	closeCh         chan struct{}
	cleanupHook     func() // Test hook used for testing
}

func newNodeDB(db ethdb.Database, clock mclock.Clock) *nodeDB {
	pcache, _ := lru.New(posBalanceCacheLimit)
	ncache, _ := lru.New(negBalanceCacheLimit)
	ndb := &nodeDB{
		db:      db,
		pcache:  pcache,
		ncache:  ncache,
		auxbuf:  make([]byte, 37),
		clock:   clock,
		closeCh: make(chan struct{}),
	}
	binary.BigEndian.PutUint16(ndb.verbuf[:], uint16(nodeDBVersion))
	go ndb.expirer()
	return ndb
}

func (db *nodeDB) close() {
	close(db.closeCh)
}

func (db *nodeDB) key(id []byte, neg bool) []byte {
	prefix := positiveBalancePrefix
	if neg {
		prefix = negativeBalancePrefix
	}
	if len(prefix)+len(db.verbuf)+len(id) > len(db.auxbuf) {
		db.auxbuf = append(db.auxbuf, make([]byte, len(prefix)+len(db.verbuf)+len(id)-len(db.auxbuf))...)
	}
	copy(db.auxbuf[:len(db.verbuf)], db.verbuf[:])
	copy(db.auxbuf[len(db.verbuf):len(db.verbuf)+len(prefix)], prefix)
	copy(db.auxbuf[len(prefix)+len(db.verbuf):len(prefix)+len(db.verbuf)+len(id)], id)
	return db.auxbuf[:len(prefix)+len(db.verbuf)+len(id)]
}

func (db *nodeDB) getCumulativeTime() int64 {
	blob, err := db.db.Get(append(cumulativeRunningTimeKey, db.verbuf[:]...))
	if err != nil || len(blob) == 0 {
		return 0
	}
	return int64(binary.BigEndian.Uint64(blob))
}

func (db *nodeDB) setCumulativeTime(v int64) {
	binary.BigEndian.PutUint64(db.auxbuf[:8], uint64(v))
	db.db.Put(append(cumulativeRunningTimeKey, db.verbuf[:]...), db.auxbuf[:8])
}

func (db *nodeDB) getOrNewPB(id enode.ID) posBalance {
	key := db.key(id.Bytes(), false)
	item, exist := db.pcache.Get(string(key))
	if exist {
		return item.(posBalance)
	}
	var balance posBalance
	if enc, err := db.db.Get(key); err == nil {
		if err := rlp.DecodeBytes(enc, &balance); err != nil {
			log.Error("Failed to decode positive balance", "err", err)
		}
	}
	db.pcache.Add(string(key), balance)
	return balance
}

func (db *nodeDB) setPB(id enode.ID, b posBalance) {
	if b.value == 0 && len(b.meta) == 0 {
		db.delPB(id)
		return
	}
	key := db.key(id.Bytes(), false)
	enc, err := rlp.EncodeToBytes(&(b))
	if err != nil {
		log.Error("Failed to encode positive balance", "err", err)
		return
	}
	db.db.Put(key, enc)
	db.pcache.Add(string(key), b)
}

func (db *nodeDB) delPB(id enode.ID) {
	key := db.key(id.Bytes(), false)
	db.db.Delete(key)
	db.pcache.Remove(string(key))
}

// getPosBalanceIDs returns a lexicographically ordered list of IDs of accounts
// with a positive balance
func (db *nodeDB) getPosBalanceIDs(start, stop enode.ID, maxCount int) (result []enode.ID) {
	if maxCount <= 0 {
		return
	}
	it := db.db.NewIteratorWithStart(db.key(start.Bytes(), false))
	defer it.Release()
	for i := len(stop[:]) - 1; i >= 0; i-- {
		stop[i]--
		if stop[i] != 255 {
			break
		}
	}
	stopKey := db.key(stop.Bytes(), false)
	keyLen := len(stopKey)

	for it.Next() {
		var id enode.ID
		if len(it.Key()) != keyLen || bytes.Compare(it.Key(), stopKey) == 1 {
			return
		}
		copy(id[:], it.Key()[keyLen-len(id):])
		result = append(result, id)
		if len(result) == maxCount {
			return
		}
	}
	return
}

func (db *nodeDB) getOrNewNB(id string) negBalance {
	key := db.key([]byte(id), true)
	item, exist := db.ncache.Get(string(key))
	if exist {
		return item.(negBalance)
	}
	var balance negBalance
	if enc, err := db.db.Get(key); err == nil {
		if err := rlp.DecodeBytes(enc, &balance); err != nil {
			log.Error("Failed to decode negative balance", "err", err)
		}
	}
	db.ncache.Add(string(key), balance)
	return balance
}

func (db *nodeDB) setNB(id string, b negBalance) {
	key := db.key([]byte(id), true)
	enc, err := rlp.EncodeToBytes(&(b))
	if err != nil {
		log.Error("Failed to encode negative balance", "err", err)
		return
	}
	db.db.Put(key, enc)
	db.ncache.Add(string(key), b)
}

func (db *nodeDB) delNB(id string) {
	key := db.key([]byte(id), true)
	db.db.Delete(key)
	db.ncache.Remove(string(key))
}

func (db *nodeDB) expirer() {
	for {
		select {
		case <-db.clock.After(dbCleanupCycle):
			db.expireNodes()
		case <-db.closeCh:
			return
		}
	}
}

// expireNodes iterates the whole node db and checks whether the negative balance
// entry can deleted.
//
// The rationale behind this is: server doesn't need to keep the negative balance
// records if they are low enough.
func (db *nodeDB) expireNodes() {
	var (
		visited int
		deleted int
		start   = time.Now()
	)
	iter := db.db.NewIteratorWithPrefix(append(db.verbuf[:], negativeBalancePrefix...))
	for iter.Next() {
		visited += 1
		var balance negBalance
		if err := rlp.DecodeBytes(iter.Value(), &balance); err != nil {
			log.Error("Failed to decode negative balance", "err", err)
			continue
		}
		if db.nbEvictCallBack != nil && db.nbEvictCallBack(db.clock.Now(), balance) {
			deleted += 1
			db.db.Delete(iter.Key())
		}
	}
	// Invoke testing hook if it's not nil.
	if db.cleanupHook != nil {
		db.cleanupHook()
	}
	log.Debug("Expire nodes", "visited", visited, "deleted", deleted, "elapsed", common.PrettyDuration(time.Since(start)))
}
