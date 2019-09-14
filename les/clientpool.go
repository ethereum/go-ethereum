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
	"io"
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	negBalanceExpTC      = time.Hour        // time constant for exponentially reducing negative balance
	fixedPointMultiplier = 0x1000000        // constant to convert logarithms to fixed point format
	connectedBias        = time.Minute * 5  // this bias is applied in favor of already connected clients in order to avoid kicking them out very soon
	lazyQueueRefresh     = time.Second * 10 // refresh period of the connected queue
)

var (
	clientPoolDbKey    = []byte("clientPool")
	clientBalanceDbKey = []byte("clientPool-balance")
)

// clientPool implements a client database that assigns a priority to each client
// based on a positive and negative balance. Positive balance is externally assigned
// to prioritized clients and is decreased with connection time and processed
// requests (unless the price factors are zero). If the positive balance is zero
// then negative balance is accumulated. Balance tracking and priority calculation
// for connected clients is done by balanceTracker. connectedQueue ensures that
// clients with the lowest positive or highest negative balance get evicted when
// the total capacity allowance is full and new clients with a better balance want
// to connect. Already connected nodes receive a small bias in their favor in order
// to avoid accepting and instantly kicking out clients.
// Balances of disconnected clients are stored in posBalanceQueue and negBalanceQueue
// and are also saved in the database. Negative balance is transformed into a
// logarithmic form with a constantly shifting linear offset in order to implement
// an exponential decrease. negBalanceQueue has a limited size and drops the smallest
// values when necessary. Positive balances are stored in the database as long as
// they exist, posBalanceQueue only acts as a cache for recently accessed entries.
type clientPool struct {
	db         ethdb.Database
	lock       sync.Mutex
	clock      mclock.Clock
	stopCh     chan chan struct{}
	closed     bool
	removePeer func(enode.ID)

	queueLimit, countLimit                          int
	freeClientCap, capacityLimit, connectedCapacity uint64

	connectedMap                     map[enode.ID]*clientInfo
	posBalanceMap                    map[enode.ID]*posBalance
	negBalanceMap                    map[string]*negBalance
	connectedQueue                   *prque.LazyQueue
	posBalanceQueue, negBalanceQueue *prque.Prque
	posFactors, negFactors           priceFactors
	posBalanceAccessCounter          int64
	startupTime                      mclock.AbsTime
	logOffsetAtStartup               int64
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
}

// clientInfo represents a connected client
type clientInfo struct {
	address        string
	id             enode.ID
	capacity       uint64
	priority       bool
	pool           *clientPool
	peer           clientPeer
	queueIndex     int // position in connectedQueue
	balanceTracker balanceTracker
}

// connSetIndex callback updates clientInfo item index in connectedQueue
func connSetIndex(a interface{}, index int) {
	a.(*clientInfo).queueIndex = index
}

// connPriority callback returns actual priority of clientInfo item in connectedQueue
func connPriority(a interface{}, now mclock.AbsTime) int64 {
	c := a.(*clientInfo)
	return c.balanceTracker.getPriority(now)
}

// connMaxPriority callback returns estimated maximum priority of clientInfo item in connectedQueue
func connMaxPriority(a interface{}, until mclock.AbsTime) int64 {
	c := a.(*clientInfo)
	pri := c.balanceTracker.estimatedPriority(until, true)
	c.balanceTracker.addCallback(balanceCallbackQueue, pri+1, func() {
		c.pool.lock.Lock()
		if c.queueIndex != -1 {
			c.pool.connectedQueue.Update(c.queueIndex)
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
func newClientPool(db ethdb.Database, freeClientCap uint64, queueLimit int, clock mclock.Clock, removePeer func(enode.ID)) *clientPool {
	pool := &clientPool{
		db:              db,
		clock:           clock,
		connectedMap:    make(map[enode.ID]*clientInfo),
		posBalanceMap:   make(map[enode.ID]*posBalance),
		negBalanceMap:   make(map[string]*negBalance),
		connectedQueue:  prque.NewLazyQueue(connSetIndex, connPriority, connMaxPriority, clock, lazyQueueRefresh),
		negBalanceQueue: prque.New(negSetIndex),
		posBalanceQueue: prque.New(posSetIndex),
		freeClientCap:   freeClientCap,
		queueLimit:      queueLimit,
		removePeer:      removePeer,
		stopCh:          make(chan chan struct{}),
	}
	pool.loadFromDb()
	go func() {
		for {
			select {
			case <-clock.After(lazyQueueRefresh):
				pool.lock.Lock()
				pool.connectedQueue.Refresh()
				pool.lock.Unlock()
			case stop := <-pool.stopCh:
				close(stop)
				return
			}
		}
	}()
	return pool
}

// stop shuts the client pool down
func (f *clientPool) stop() {
	stop := make(chan struct{})
	f.stopCh <- stop
	<-stop
	f.lock.Lock()
	f.closed = true
	f.saveToDb()
	f.lock.Unlock()
}

// connect should be called after a successful handshake. If the connection was
// rejected, there is no need to call disconnect.
func (f *clientPool) connect(peer clientPeer, capacity uint64) bool {
	f.lock.Lock()
	defer f.lock.Unlock()

	// Short circuit is clientPool is already closed.
	if f.closed {
		return false
	}
	// Dedup connected peers.
	id, freeID := peer.ID(), peer.freeClientId()
	if _, ok := f.connectedMap[id]; ok {
		clientRejectedMeter.Mark(1)
		log.Debug("Client already connected", "address", freeID, "id", peerIdToString(id))
		return false
	}
	// Create a clientInfo but do not add it yet
	now := f.clock.Now()
	posBalance := f.getPosBalance(id).value
	e := &clientInfo{pool: f, peer: peer, address: freeID, queueIndex: -1, id: id, priority: posBalance != 0}

	var negBalance uint64
	nb := f.negBalanceMap[freeID]
	if nb != nil {
		negBalance = uint64(math.Exp(float64(nb.logValue-f.logOffset(now)) / fixedPointMultiplier))
	}
	// If the client is a free client, assign with a low free capacity,
	// Otherwise assign with the given value(priority client)
	if !e.priority {
		capacity = f.freeClientCap
	}
	// Ensure the capacity will never lower than the free capacity.
	if capacity < f.freeClientCap {
		capacity = f.freeClientCap
	}
	e.capacity = capacity

	e.balanceTracker.init(f.clock, capacity)
	e.balanceTracker.setBalance(posBalance, negBalance)
	f.setClientPriceFactors(e)

	// If the number of clients already connected in the clientpool exceeds its
	// capacity, evict some clients with lowest priority.
	//
	// If the priority of the newly added client is lower than the priority of
	// all connected clients, the client is rejected.
	newCapacity := f.connectedCapacity + capacity
	newCount := f.connectedQueue.Size() + 1
	if newCapacity > f.capacityLimit || newCount > f.countLimit {
		var (
			kickList     []*clientInfo
			kickPriority int64
		)
		f.connectedQueue.MultiPop(func(data interface{}, priority int64) bool {
			c := data.(*clientInfo)
			kickList = append(kickList, c)
			kickPriority = priority
			newCapacity -= c.capacity
			newCount--
			return newCapacity > f.capacityLimit || newCount > f.countLimit
		})
		if newCapacity > f.capacityLimit || newCount > f.countLimit || (e.balanceTracker.estimatedPriority(now+mclock.AbsTime(connectedBias), false)-kickPriority) > 0 {
			// reject client
			for _, c := range kickList {
				f.connectedQueue.Push(c)
			}
			clientRejectedMeter.Mark(1)
			log.Debug("Client rejected", "address", freeID, "id", peerIdToString(id))
			return false
		}
		// accept new client, drop old ones
		for _, c := range kickList {
			f.dropClient(c, now, true)
		}
	}
	// client accepted, finish setting it up
	if nb != nil {
		delete(f.negBalanceMap, freeID)
		f.negBalanceQueue.Remove(nb.queueIndex)
	}
	if e.priority {
		e.balanceTracker.addCallback(balanceCallbackZero, 0, func() { f.balanceExhausted(id) })
	}
	f.connectedMap[id] = e
	f.connectedQueue.Push(e)
	f.connectedCapacity += e.capacity
	totalConnectedGauge.Update(int64(f.connectedCapacity))
	if e.capacity != f.freeClientCap {
		e.peer.updateCapacity(e.capacity)
	}
	clientConnectedMeter.Mark(1)
	log.Debug("Client accepted", "address", freeID)
	return true
}

// disconnect should be called when a connection is terminated. If the disconnection
// was initiated by the pool itself using disconnectFn then calling disconnect is
// not necessary but permitted.
func (f *clientPool) disconnect(p clientPeer) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if f.closed {
		return
	}
	address := p.freeClientId()
	id := p.ID()
	// Short circuit if the peer hasn't been registered.
	e := f.connectedMap[id]
	if e == nil {
		log.Debug("Client not connected", "address", address, "id", peerIdToString(id))
		return
	}
	f.dropClient(e, f.clock.Now(), false)
}

// dropClient removes a client from the connected queue and finalizes its balance.
// If kick is true then it also initiates the disconnection.
func (f *clientPool) dropClient(e *clientInfo, now mclock.AbsTime, kick bool) {
	if _, ok := f.connectedMap[e.id]; !ok {
		return
	}
	f.finalizeBalance(e, now)
	f.connectedQueue.Remove(e.queueIndex)
	delete(f.connectedMap, e.id)
	f.connectedCapacity -= e.capacity
	totalConnectedGauge.Update(int64(f.connectedCapacity))
	if kick {
		clientKickedMeter.Mark(1)
		log.Debug("Client kicked out", "address", e.address)
		f.removePeer(e.id)
	} else {
		clientDisconnectedMeter.Mark(1)
		log.Debug("Client disconnected", "address", e.address)
	}
}

// finalizeBalance stops the balance tracker, retrieves the final balances and
// stores them in posBalanceQueue and negBalanceQueue
func (f *clientPool) finalizeBalance(c *clientInfo, now mclock.AbsTime) {
	c.balanceTracker.stop(now)
	pos, neg := c.balanceTracker.getBalance(now)
	pb := f.getPosBalance(c.id)
	pb.value = pos
	f.storePosBalance(pb)
	if neg < 1 {
		neg = 1
	}
	nb := &negBalance{address: c.address, queueIndex: -1, logValue: int64(math.Log(float64(neg))*fixedPointMultiplier) + f.logOffset(now)}
	f.negBalanceMap[c.address] = nb
	f.negBalanceQueue.Push(nb, -nb.logValue)
	if f.negBalanceQueue.Size() > f.queueLimit {
		nn := f.negBalanceQueue.PopItem().(*negBalance)
		delete(f.negBalanceMap, nn.address)
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
	c.priority = false
	if c.capacity != f.freeClientCap {
		f.connectedCapacity += f.freeClientCap - c.capacity
		totalConnectedGauge.Update(int64(f.connectedCapacity))
		c.capacity = f.freeClientCap
		c.peer.updateCapacity(c.capacity)
	}
}

// setConnLimit sets the maximum number and total capacity of connected clients,
// dropping some of them if necessary.
func (f *clientPool) setLimits(count int, totalCap uint64) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.countLimit = count
	f.capacityLimit = totalCap
	if f.connectedCapacity > f.capacityLimit || f.connectedQueue.Size() > f.countLimit {
		now := mclock.Now()
		f.connectedQueue.MultiPop(func(data interface{}, priority int64) bool {
			c := data.(*clientInfo)
			f.dropClient(c, now, true)
			return f.connectedCapacity > f.capacityLimit || f.connectedQueue.Size() > f.countLimit
		})
	}
}

// requestCost feeds request cost after serving a request from the given peer.
func (f *clientPool) requestCost(p *peer, cost uint64) {
	f.lock.Lock()
	defer f.lock.Unlock()

	info, exist := f.connectedMap[p.ID()]
	if !exist || f.closed {
		return
	}
	info.balanceTracker.requestCost(cost)
}

// logOffset calculates the time-dependent offset for the logarithmic
// representation of negative balance
func (f *clientPool) logOffset(now mclock.AbsTime) int64 {
	// Note: fixedPointMultiplier acts as a multiplier here; the reason for dividing the divisor
	// is to avoid int64 overflow. We assume that int64(negBalanceExpTC) >> fixedPointMultiplier.
	logDecay := int64((time.Duration(now - f.startupTime)) / (negBalanceExpTC / fixedPointMultiplier))
	return f.logOffsetAtStartup + logDecay
}

// setPriceFactors changes pricing factors for both positive and negative balances.
// Applies to connected clients and also future connections.
func (f *clientPool) setPriceFactors(posFactors, negFactors priceFactors) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.posFactors, f.negFactors = posFactors, negFactors
	for _, c := range f.connectedMap {
		f.setClientPriceFactors(c)
	}
}

// setClientPriceFactors sets the pricing factors for an individual connected client
func (f *clientPool) setClientPriceFactors(c *clientInfo) {
	c.balanceTracker.setFactors(true, f.negFactors.timeFactor+float64(c.capacity)*f.negFactors.capacityFactor/1000000, f.negFactors.requestFactor)
	c.balanceTracker.setFactors(false, f.posFactors.timeFactor+float64(c.capacity)*f.posFactors.capacityFactor/1000000, f.posFactors.requestFactor)
}

// clientPoolStorage is the RLP representation of the pool's database storage
type clientPoolStorage struct {
	LogOffset uint64
	List      []*negBalance
}

// loadFromDb restores pool status from the database storage
// (automatically called at initialization)
func (f *clientPool) loadFromDb() {
	enc, err := f.db.Get(clientPoolDbKey)
	if err != nil {
		return
	}
	var storage clientPoolStorage
	err = rlp.DecodeBytes(enc, &storage)
	if err != nil {
		log.Error("Failed to decode client list", "err", err)
		return
	}
	f.logOffsetAtStartup = int64(storage.LogOffset)
	f.startupTime = f.clock.Now()
	for _, e := range storage.List {
		log.Debug("Loaded free client record", "address", e.address, "logValue", e.logValue)
		f.negBalanceMap[e.address] = e
		f.negBalanceQueue.Push(e, -e.logValue)
	}
}

// saveToDb saves pool status to the database storage
// (automatically called during shutdown)
func (f *clientPool) saveToDb() {
	now := f.clock.Now()
	storage := clientPoolStorage{
		LogOffset: uint64(f.logOffset(now)),
	}
	for _, c := range f.connectedMap {
		f.finalizeBalance(c, now)
	}
	i := 0
	storage.List = make([]*negBalance, len(f.negBalanceMap))
	for _, e := range f.negBalanceMap {
		storage.List[i] = e
		i++
	}
	enc, err := rlp.EncodeToBytes(storage)
	if err != nil {
		log.Error("Failed to encode negative balance list", "err", err)
	} else {
		f.db.Put(clientPoolDbKey, enc)
	}
}

// storePosBalance stores a single positive balance entry in the database
func (f *clientPool) storePosBalance(b *posBalance) {
	if b.value == b.lastStored {
		return
	}
	enc, err := rlp.EncodeToBytes(b)
	if err != nil {
		log.Error("Failed to encode client balance", "err", err)
	} else {
		f.db.Put(append(clientBalanceDbKey, b.id[:]...), enc)
		b.lastStored = b.value
	}
}

// getPosBalance retrieves a single positive balance entry from cache or the database
func (f *clientPool) getPosBalance(id enode.ID) *posBalance {
	if b, ok := f.posBalanceMap[id]; ok {
		f.posBalanceQueue.Remove(b.queueIndex)
		f.posBalanceAccessCounter--
		f.posBalanceQueue.Push(b, f.posBalanceAccessCounter)
		return b
	}
	balance := &posBalance{}
	if enc, err := f.db.Get(append(clientBalanceDbKey, id[:]...)); err == nil {
		if err := rlp.DecodeBytes(enc, balance); err != nil {
			log.Error("Failed to decode client balance", "err", err)
			balance = &posBalance{}
		}
	}
	balance.id = id
	balance.queueIndex = -1
	if f.posBalanceQueue.Size() >= f.queueLimit {
		b := f.posBalanceQueue.PopItem().(*posBalance)
		f.storePosBalance(b)
		delete(f.posBalanceMap, b.id)
	}
	f.posBalanceAccessCounter--
	f.posBalanceQueue.Push(balance, f.posBalanceAccessCounter)
	f.posBalanceMap[id] = balance
	return balance
}

// addBalance updates the positive balance of a client.
// If setTotal is false then the given amount is added to the balance.
// If setTotal is true then amount represents the total amount ever added to the
// given ID and positive balance is increased by (amount-lastTotal) while lastTotal
// is updated to amount. This method also allows removing positive balance.
func (f *clientPool) addBalance(id enode.ID, amount uint64, setTotal bool) {
	f.lock.Lock()
	defer f.lock.Unlock()

	pb := f.getPosBalance(id)
	c := f.connectedMap[id]
	var negBalance uint64
	if c != nil {
		pb.value, negBalance = c.balanceTracker.getBalance(f.clock.Now())
	}
	if setTotal {
		if pb.value+amount > pb.lastTotal {
			pb.value += amount - pb.lastTotal
		} else {
			pb.value = 0
		}
		pb.lastTotal = amount
	} else {
		pb.value += amount
		pb.lastTotal += amount
	}
	f.storePosBalance(pb)
	if c != nil {
		c.balanceTracker.setBalance(pb.value, negBalance)
		if !c.priority && pb.value > 0 {
			c.priority = true
			c.balanceTracker.addCallback(balanceCallbackZero, 0, func() { f.balanceExhausted(id) })
		}
	}
}

// posBalance represents a recently accessed positive balance entry
type posBalance struct {
	id                           enode.ID
	value, lastStored, lastTotal uint64
	queueIndex                   int // position in posBalanceQueue
}

// EncodeRLP implements rlp.Encoder
func (e *posBalance) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{e.value, e.lastTotal})
}

// DecodeRLP implements rlp.Decoder
func (e *posBalance) DecodeRLP(s *rlp.Stream) error {
	var entry struct {
		Value, LastTotal uint64
	}
	if err := s.Decode(&entry); err != nil {
		return err
	}
	e.value = entry.Value
	e.lastStored = entry.Value
	e.lastTotal = entry.LastTotal
	return nil
}

// posSetIndex callback updates posBalance item index in posBalanceQueue
func posSetIndex(a interface{}, index int) {
	a.(*posBalance).queueIndex = index
}

// negBalance represents a negative balance entry of a disconnected client
type negBalance struct {
	address    string
	logValue   int64
	queueIndex int // position in negBalanceQueue
}

// EncodeRLP implements rlp.Encoder
func (e *negBalance) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{e.address, uint64(e.logValue)})
}

// DecodeRLP implements rlp.Decoder
func (e *negBalance) DecodeRLP(s *rlp.Stream) error {
	var entry struct {
		Address  string
		LogValue uint64
	}
	if err := s.Decode(&entry); err != nil {
		return err
	}
	e.address = entry.Address
	e.logValue = int64(entry.LogValue)
	e.queueIndex = -1
	return nil
}

// negSetIndex callback updates negBalance item index in negBalanceQueue
func negSetIndex(a interface{}, index int) {
	a.(*negBalance).queueIndex = index
}
