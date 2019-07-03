// Copyright 2016 The go-ethereum Authors
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
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

// freeClientPool implements a client database that limits the connection time
// of each client and manages accepting/rejecting incoming connections and even
// kicking out some connected clients. The pool calculates recent usage time
// for each known client (a value that increases linearly when the client is
// connected and decreases exponentially when not connected). Clients with lower
// recent usage are preferred, unknown nodes have the highest priority. Already
// connected nodes receive a small bias in their favor in order to avoid accepting
// and instantly kicking out clients.
//
// Note: the pool can use any string for client identification. Using signature
// keys for that purpose would not make sense when being known has a negative
// value for the client. Currently the LES protocol manager uses IP addresses
// (without port address) to identify clients.
type freeClientPool struct {
	db         ethdb.Database
	lock       sync.Mutex
	clock      mclock.Clock
	closed     bool
	removePeer func(string)

	connectedLimit, totalLimit int
	freeClientCap              uint64
	connectedCap               uint64

	addressMap            map[string]*freeClientPoolEntry
	connPool, disconnPool *prque.Prque
	startupTime           mclock.AbsTime
	logOffsetAtStartup    int64
}

const (
	recentUsageExpTC     = time.Hour   // time constant of the exponential weighting window for "recent" server usage
	fixedPointMultiplier = 0x1000000   // constant to convert logarithms to fixed point format
	connectedBias        = time.Minute // this bias is applied in favor of already connected clients in order to avoid kicking them out very soon
)

// newFreeClientPool creates a new free client pool
func newFreeClientPool(db ethdb.Database, freeClientCap uint64, totalLimit int, clock mclock.Clock, removePeer func(string)) *freeClientPool {
	pool := &freeClientPool{
		db:            db,
		clock:         clock,
		addressMap:    make(map[string]*freeClientPoolEntry),
		connPool:      prque.New(poolSetIndex),
		disconnPool:   prque.New(poolSetIndex),
		freeClientCap: freeClientCap,
		totalLimit:    totalLimit,
		removePeer:    removePeer,
	}
	pool.loadFromDb()
	return pool
}

func (f *freeClientPool) stop() {
	f.lock.Lock()
	f.closed = true
	f.saveToDb()
	f.lock.Unlock()
}

// freeClientId returns a string identifier for the peer. Multiple peers with the
// same identifier can not be in the free client pool simultaneously.
func freeClientId(p *peer) string {
	if addr, ok := p.RemoteAddr().(*net.TCPAddr); ok {
		if addr.IP.IsLoopback() {
			// using peer id instead of loopback ip address allows multiple free
			// connections from local machine to own server
			return p.id
		} else {
			return addr.IP.String()
		}
	}
	return ""
}

// registerPeer implements clientPool
func (f *freeClientPool) registerPeer(p *peer) {
	if freeId := freeClientId(p); freeId != "" {
		if !f.connect(freeId, p.id) {
			f.removePeer(p.id)
		}
	}
}

// connect should be called after a successful handshake. If the connection was
// rejected, there is no need to call disconnect.
func (f *freeClientPool) connect(address, id string) bool {
	f.lock.Lock()
	defer f.lock.Unlock()

	if f.closed {
		return false
	}
	if f.connectedLimit == 0 {
		log.Debug("Client rejected", "address", address)
		return false
	}
	e := f.addressMap[address]
	now := f.clock.Now()
	var recentUsage int64
	if e == nil {
		e = &freeClientPoolEntry{address: address, index: -1, id: id}
		f.addressMap[address] = e
	} else {
		if e.connected {
			log.Debug("Client already connected", "address", address)
			return false
		}
		recentUsage = int64(math.Exp(float64(e.logUsage-f.logOffset(now)) / fixedPointMultiplier))
	}
	e.linUsage = recentUsage - int64(now)
	// check whether (linUsage+connectedBias) is smaller than the highest entry in the connected pool
	if f.connPool.Size() == f.connectedLimit {
		i := f.connPool.PopItem().(*freeClientPoolEntry)
		if e.linUsage+int64(connectedBias)-i.linUsage < 0 {
			// kick it out and accept the new client
			f.dropClient(i, now)
			clientKickedMeter.Mark(1)
			f.connectedCap -= f.freeClientCap
		} else {
			// keep the old client and reject the new one
			f.connPool.Push(i, i.linUsage)
			log.Debug("Client rejected", "address", address)
			clientRejectedMeter.Mark(1)
			return false
		}
	}
	f.disconnPool.Remove(e.index)
	e.connected = true
	e.id = id
	f.connPool.Push(e, e.linUsage)
	if f.connPool.Size()+f.disconnPool.Size() > f.totalLimit {
		f.disconnPool.Pop()
	}
	f.connectedCap += f.freeClientCap
	totalConnectedGauge.Update(int64(f.connectedCap))
	clientConnectedMeter.Mark(1)
	log.Debug("Client accepted", "address", address)
	return true
}

// unregisterPeer implements clientPool
func (f *freeClientPool) unregisterPeer(p *peer) {
	if freeId := freeClientId(p); freeId != "" {
		f.disconnect(freeId)
	}
}

// disconnect should be called when a connection is terminated. If the disconnection
// was initiated by the pool itself using disconnectFn then calling disconnect is
// not necessary but permitted.
func (f *freeClientPool) disconnect(address string) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if f.closed {
		return
	}
	// Short circuit if the peer hasn't been registered.
	e := f.addressMap[address]
	if e == nil {
		return
	}
	now := f.clock.Now()
	if !e.connected {
		log.Debug("Client already disconnected", "address", address)
		return
	}
	f.connPool.Remove(e.index)
	f.calcLogUsage(e, now)
	e.connected = false
	f.disconnPool.Push(e, -e.logUsage)
	f.connectedCap -= f.freeClientCap
	totalConnectedGauge.Update(int64(f.connectedCap))
	log.Debug("Client disconnected", "address", address)
}

// setConnLimit sets the maximum number of free client slots and also drops
// some peers if necessary
func (f *freeClientPool) setLimits(count int, totalCap uint64) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.connectedLimit = int(totalCap / f.freeClientCap)
	if count < f.connectedLimit {
		f.connectedLimit = count
	}
	now := mclock.Now()
	for f.connPool.Size() > f.connectedLimit {
		i := f.connPool.PopItem().(*freeClientPoolEntry)
		f.dropClient(i, now)
		f.connectedCap -= f.freeClientCap
	}
	totalConnectedGauge.Update(int64(f.connectedCap))
}

// dropClient disconnects a client and also moves it from the connected to the
// disconnected pool
func (f *freeClientPool) dropClient(i *freeClientPoolEntry, now mclock.AbsTime) {
	f.connPool.Remove(i.index)
	f.calcLogUsage(i, now)
	i.connected = false
	f.disconnPool.Push(i, -i.logUsage)
	log.Debug("Client kicked out", "address", i.address)
	f.removePeer(i.id)
}

// logOffset calculates the time-dependent offset for the logarithmic
// representation of recent usage
func (f *freeClientPool) logOffset(now mclock.AbsTime) int64 {
	// Note: fixedPointMultiplier acts as a multiplier here; the reason for dividing the divisor
	// is to avoid int64 overflow. We assume that int64(recentUsageExpTC) >> fixedPointMultiplier.
	logDecay := int64((time.Duration(now - f.startupTime)) / (recentUsageExpTC / fixedPointMultiplier))
	return f.logOffsetAtStartup + logDecay
}

// calcLogUsage converts recent usage from linear to logarithmic representation
// when disconnecting a peer or closing the client pool
func (f *freeClientPool) calcLogUsage(e *freeClientPoolEntry, now mclock.AbsTime) {
	dt := e.linUsage + int64(now)
	if dt < 1 {
		dt = 1
	}
	e.logUsage = int64(math.Log(float64(dt))*fixedPointMultiplier) + f.logOffset(now)
}

// freeClientPoolStorage is the RLP representation of the pool's database storage
type freeClientPoolStorage struct {
	LogOffset uint64
	List      []*freeClientPoolEntry
}

// loadFromDb restores pool status from the database storage
// (automatically called at initialization)
func (f *freeClientPool) loadFromDb() {
	enc, err := f.db.Get([]byte("freeClientPool"))
	if err != nil {
		return
	}
	var storage freeClientPoolStorage
	err = rlp.DecodeBytes(enc, &storage)
	if err != nil {
		log.Error("Failed to decode client list", "err", err)
		return
	}
	f.logOffsetAtStartup = int64(storage.LogOffset)
	f.startupTime = f.clock.Now()
	for _, e := range storage.List {
		log.Debug("Loaded free client record", "address", e.address, "logUsage", e.logUsage)
		f.addressMap[e.address] = e
		f.disconnPool.Push(e, -e.logUsage)
	}
}

// saveToDb saves pool status to the database storage
// (automatically called during shutdown)
func (f *freeClientPool) saveToDb() {
	now := f.clock.Now()
	storage := freeClientPoolStorage{
		LogOffset: uint64(f.logOffset(now)),
		List:      make([]*freeClientPoolEntry, len(f.addressMap)),
	}
	i := 0
	for _, e := range f.addressMap {
		if e.connected {
			f.calcLogUsage(e, now)
		}
		storage.List[i] = e
		i++
	}
	enc, err := rlp.EncodeToBytes(storage)
	if err != nil {
		log.Error("Failed to encode client list", "err", err)
	} else {
		f.db.Put([]byte("freeClientPool"), enc)
	}
}

// freeClientPoolEntry represents a client address known by the pool.
// When connected, recent usage is calculated as linUsage + int64(clock.Now())
// When disconnected, it is calculated as exp(logUsage - logOffset) where logOffset
// also grows linearly with time while the server is running.
// Conversion between linear and logarithmic representation happens when connecting
// or disconnecting the node.
//
// Note: linUsage and logUsage are values used with constantly growing offsets so
// even though they are close to each other at any time they may wrap around int64
// limits over time. Comparison should be performed accordingly.
type freeClientPoolEntry struct {
	address, id        string
	connected          bool
	disconnectFn       func()
	linUsage, logUsage int64
	index              int
}

func (e *freeClientPoolEntry) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{e.address, uint64(e.logUsage)})
}

func (e *freeClientPoolEntry) DecodeRLP(s *rlp.Stream) error {
	var entry struct {
		Address  string
		LogUsage uint64
	}
	if err := s.Decode(&entry); err != nil {
		return err
	}
	e.address = entry.Address
	e.logUsage = int64(entry.LogUsage)
	e.connected = false
	e.index = -1
	return nil
}

// poolSetIndex callback is used by both priority queues to set/update the index of
// the element in the queue. Index is needed to remove elements other than the top one.
func poolSetIndex(a interface{}, i int) {
	a.(*freeClientPoolEntry).index = i
}
