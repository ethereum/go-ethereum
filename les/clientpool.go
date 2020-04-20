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
	persistCumulativeTimeRefresh = time.Minute * 5  // refresh period of the cumulative running time persistence
	posBalanceCacheLimit         = 8192             // the maximum number of cached items in positive balance queue
	negBalanceCacheLimit         = 8192             // the maximum number of cached items in negative balance queue

	// connectedBias is applied to already connected clients So that
	// already connected client won't be kicked out very soon and we
	// can ensure all connected clients can have enough time to request
	// or sync some data.
	//
	// todo(rjl493456442) make it configurable. It can be the option of
	// free trial time!
	connectedBias = time.Minute * 3
)

// clientPool implements a client database that assigns a priority to each client
// based on a positive and negative balance. Positive balance is externally assigned
// to prioritized clients and is decreased with connection time and processed
// requests (unless the price factors are zero). If the positive balance is zero
// then negative balance is accumulated.
//
// Balance tracking and priority calculation for connected clients is done by
// balanceTracker. connectedQueue ensures that clients with the lowest positive or
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

	connectedMap   map[enode.ID]*clientInfo
	connectedQueue *prque.LazyQueue

	defaultPosFactors, defaultNegFactors priceFactors

	connLimit         int            // The maximum number of connections that clientpool can support
	capLimit          uint64         // The maximum cumulative capacity that clientpool can support
	connectedCap      uint64         // The sum of the capacity of the current clientpool connected
	priorityConnected uint64         // The sum of the capacity of currently connected priority clients
	freeClientCap     uint64         // The capacity value of each free client
	startTime         mclock.AbsTime // The timestamp at which the clientpool started running
	cumulativeTime    int64          // The cumulative running time of clientpool at the start point.
	disableBias       bool           // Disable connection bias(used in testing)
}

// clientPoolPeer represents a client peer in the pool.
// Positive balances are assigned to node key while negative balances are assigned
// to freeClientId. Currently network IP address without port is used because
// clients have a limited access to IP addresses while new node keys can be easily
// generated so it would be useless to assign a negative value to them.
type clientPoolPeer interface {
	ID() enode.ID
	freeClientId() string
	updateCapacity(uint64)
	freezeClient()
}

// clientInfo represents a connected client
type clientInfo struct {
	address                string
	id                     enode.ID
	connectedAt            mclock.AbsTime
	capacity               uint64
	priority               bool
	pool                   *clientPool
	peer                   clientPoolPeer
	queueIndex             int // position in connectedQueue
	balanceTracker         balanceTracker
	posFactors, negFactors priceFactors
	balanceMetaInfo        string
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
func newClientPool(db ethdb.Database, freeClientCap uint64, clock mclock.Clock, removePeer func(enode.ID)) *clientPool {
	ndb := newNodeDB(db, clock)
	pool := &clientPool{
		ndb:            ndb,
		clock:          clock,
		connectedMap:   make(map[enode.ID]*clientInfo),
		connectedQueue: prque.NewLazyQueue(connSetIndex, connPriority, connMaxPriority, clock, lazyQueueRefresh),
		freeClientCap:  freeClientCap,
		removePeer:     removePeer,
		startTime:      clock.Now(),
		cumulativeTime: ndb.getCumulativeTime(),
		stopCh:         make(chan struct{}),
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
				pool.connectedQueue.Refresh()
				pool.lock.Unlock()
			case <-clock.After(persistCumulativeTimeRefresh):
				pool.ndb.setCumulativeTime(pool.logOffset(clock.Now()))
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

// connect should be called after a successful handshake. If the connection was
// rejected, there is no need to call disconnect.
func (f *clientPool) connect(peer clientPoolPeer, capacity uint64) bool {
	f.lock.Lock()
	defer f.lock.Unlock()

	// Short circuit if clientPool is already closed.
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
	var (
		posBalance uint64
		negBalance uint64
		now        = f.clock.Now()
	)
	pb := f.ndb.getOrNewPB(id)
	posBalance = pb.value

	nb := f.ndb.getOrNewNB(freeID)
	if nb.logValue != 0 {
		negBalance = uint64(math.Exp(float64(nb.logValue-f.logOffset(now))/fixedPointMultiplier) * float64(time.Second))
	}
	e := &clientInfo{
		pool:            f,
		peer:            peer,
		address:         freeID,
		queueIndex:      -1,
		id:              id,
		connectedAt:     now,
		priority:        posBalance != 0,
		posFactors:      f.defaultPosFactors,
		negFactors:      f.defaultNegFactors,
		balanceMetaInfo: pb.meta,
	}
	// If the client is a free client, assign with a low free capacity,
	// Otherwise assign with the given value(priority client)
	if !e.priority || capacity == 0 {
		capacity = f.freeClientCap
	}
	e.capacity = capacity

	// Starts a balance tracker
	e.balanceTracker.init(f.clock, capacity)
	e.balanceTracker.setBalance(posBalance, negBalance)
	e.updatePriceFactors()

	// If the number of clients already connected in the clientpool exceeds its
	// capacity, evict some clients with lowest priority.
	//
	// If the priority of the newly added client is lower than the priority of
	// all connected clients, the client is rejected.
	newCapacity := f.connectedCap + capacity
	newCount := f.connectedQueue.Size() + 1
	if newCapacity > f.capLimit || newCount > f.connLimit {
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
			return newCapacity > f.capLimit || newCount > f.connLimit
		})
		bias := connectedBias
		if f.disableBias {
			bias = 0
		}
		if newCapacity > f.capLimit || newCount > f.connLimit || (e.balanceTracker.estimatedPriority(now+mclock.AbsTime(bias), false)-kickPriority) > 0 {
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

	// Register new client to connection queue.
	f.connectedMap[id] = e
	f.connectedQueue.Push(e)
	f.connectedCap += e.capacity

	// If the current client is a paid client, monitor the status of client,
	// downgrade it to normal client if positive balance is used up.
	if e.priority {
		f.priorityConnected += capacity
		e.balanceTracker.addCallback(balanceCallbackZero, 0, func() { f.balanceExhausted(id) })
	}
	// If the capacity of client is not the default value(free capacity), notify
	// it to update capacity.
	if e.capacity != f.freeClientCap {
		e.peer.updateCapacity(e.capacity)
	}
	totalConnectedGauge.Update(int64(f.connectedCap))
	clientConnectedMeter.Mark(1)
	log.Debug("Client accepted", "address", freeID)
	return true
}

// disconnect should be called when a connection is terminated. If the disconnection
// was initiated by the pool itself using disconnectFn then calling disconnect is
// not necessary but permitted.
func (f *clientPool) disconnect(p clientPoolPeer) {
	f.lock.Lock()
	defer f.lock.Unlock()

	// Short circuit if client pool is already closed.
	if f.closed {
		return
	}
	// Short circuit if the peer hasn't been registered.
	e := f.connectedMap[p.ID()]
	if e == nil {
		log.Debug("Client not connected", "address", p.freeClientId(), "id", peerIdToString(p.ID()))
		return
	}
	f.dropClient(e, f.clock.Now(), false)
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

// dropClient removes a client from the connected queue and finalizes its balance.
// If kick is true then it also initiates the disconnection.
func (f *clientPool) dropClient(e *clientInfo, now mclock.AbsTime, kick bool) {
	if _, ok := f.connectedMap[e.id]; !ok {
		return
	}
	f.finalizeBalance(e, now)
	f.connectedQueue.Remove(e.queueIndex)
	delete(f.connectedMap, e.id)
	f.connectedCap -= e.capacity
	if e.priority {
		f.priorityConnected -= e.capacity
	}
	totalConnectedGauge.Update(int64(f.connectedCap))
	if kick {
		clientKickedMeter.Mark(1)
		log.Debug("Client kicked out", "address", e.address)
		f.removePeer(e.id)
	} else {
		clientDisconnectedMeter.Mark(1)
		log.Debug("Client disconnected", "address", e.address)
	}
}

// capacityInfo returns the total capacity allowance, the total capacity of connected
// clients and the total capacity of connected and prioritized clients
func (f *clientPool) capacityInfo() (uint64, uint64, uint64) {
	f.lock.Lock()
	defer f.lock.Unlock()

	return f.capLimit, f.connectedCap, f.priorityConnected
}

// finalizeBalance stops the balance tracker, retrieves the final balances and
// stores them in posBalanceQueue and negBalanceQueue
func (f *clientPool) finalizeBalance(c *clientInfo, now mclock.AbsTime) {
	c.balanceTracker.stop(now)
	pos, neg := c.balanceTracker.getBalance(now)

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
		f.priorityConnected -= c.capacity
	}
	c.priority = false
	if c.capacity != f.freeClientCap {
		f.connectedCap += f.freeClientCap - c.capacity
		totalConnectedGauge.Update(int64(f.connectedCap))
		c.capacity = f.freeClientCap
		c.balanceTracker.setCapacity(c.capacity)
		c.peer.updateCapacity(c.capacity)
	}
	pb := f.ndb.getOrNewPB(id)
	pb.value = 0
	f.ndb.setPB(id, pb)
}

// setConnLimit sets the maximum number and total capacity of connected clients,
// dropping some of them if necessary.
func (f *clientPool) setLimits(totalConn int, totalCap uint64) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.connLimit = totalConn
	f.capLimit = totalCap
	if f.connectedCap > f.capLimit || f.connectedQueue.Size() > f.connLimit {
		f.connectedQueue.MultiPop(func(data interface{}, priority int64) bool {
			f.dropClient(data.(*clientInfo), mclock.Now(), true)
			return f.connectedCap > f.capLimit || f.connectedQueue.Size() > f.connLimit
		})
	}
}

// setCapacity sets the assigned capacity of a connected client
func (f *clientPool) setCapacity(c *clientInfo, capacity uint64) error {
	if f.connectedMap[c.id] != c {
		return fmt.Errorf("client %064x is not connected", c.id[:])
	}
	if c.capacity == capacity {
		return nil
	}
	if !c.priority {
		return errNoPriority
	}
	oldCapacity := c.capacity
	c.capacity = capacity
	f.connectedCap += capacity - oldCapacity
	c.balanceTracker.setCapacity(capacity)
	f.connectedQueue.Update(c.queueIndex)
	if f.connectedCap > f.capLimit {
		var kickList []*clientInfo
		kick := true
		f.connectedQueue.MultiPop(func(data interface{}, priority int64) bool {
			client := data.(*clientInfo)
			kickList = append(kickList, client)
			f.connectedCap -= client.capacity
			if client == c {
				kick = false
			}
			return kick && (f.connectedCap > f.capLimit)
		})
		if kick {
			now := mclock.Now()
			for _, c := range kickList {
				f.dropClient(c, now, true)
			}
		} else {
			c.capacity = oldCapacity
			c.balanceTracker.setCapacity(oldCapacity)
			for _, c := range kickList {
				f.connectedCap += c.capacity
				f.connectedQueue.Push(c)
			}
			return errNoPriority
		}
	}
	totalConnectedGauge.Update(int64(f.connectedCap))
	f.priorityConnected += capacity - oldCapacity
	c.updatePriceFactors()
	c.peer.updateCapacity(c.capacity)
	return nil
}

// requestCost feeds request cost after serving a request from the given peer.
func (f *clientPool) requestCost(p *clientPeer, cost uint64) {
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
//
// From another point of view, the result returned by the function represents
// the total time that the clientpool is cumulatively running(total_hours/multiplier).
func (f *clientPool) logOffset(now mclock.AbsTime) int64 {
	// Note: fixedPointMultiplier acts as a multiplier here; the reason for dividing the divisor
	// is to avoid int64 overflow. We assume that int64(negBalanceExpTC) >> fixedPointMultiplier.
	cumulativeTime := int64((time.Duration(now - f.startTime)) / (negBalanceExpTC / fixedPointMultiplier))
	return f.cumulativeTime + cumulativeTime
}

// setClientPriceFactors sets the pricing factors for an individual connected client
func (c *clientInfo) updatePriceFactors() {
	c.balanceTracker.setFactors(true, c.negFactors.timeFactor+float64(c.capacity)*c.negFactors.capacityFactor/1000000, c.negFactors.requestFactor)
	c.balanceTracker.setFactors(false, c.posFactors.timeFactor+float64(c.capacity)*c.posFactors.capacityFactor/1000000, c.posFactors.requestFactor)
}

// getPosBalance retrieves a single positive balance entry from cache or the database
func (f *clientPool) getPosBalance(id enode.ID) posBalance {
	f.lock.Lock()
	defer f.lock.Unlock()

	return f.ndb.getOrNewPB(id)
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
		if !c.priority && pb.value > 0 {
			// The capacity should be adjusted based on the requirement,
			// but we have no idea about the new capacity, need a second
			// call to udpate it.
			c.priority = true
			f.priorityConnected += c.capacity
			c.balanceTracker.addCallback(balanceCallbackZero, 0, func() { f.balanceExhausted(id) })
		}
		// if balance is set to zero then reverting to non-priority status
		// is handled by the balanceExhausted callback
		c.balanceMetaInfo = meta
	}
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

func (db *nodeDB) getPrefix(neg bool) []byte {
	prefix := positiveBalancePrefix
	if neg {
		prefix = negativeBalancePrefix
	}
	return append(db.verbuf[:], prefix...)
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
	prefix := db.getPrefix(false)
	it := db.db.NewIterator(prefix, start.Bytes())
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
		prefix  = db.getPrefix(true)
	)
	iter := db.db.NewIterator(prefix, nil)
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
