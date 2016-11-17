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

// Package les implements the Light Ethereum Subprotocol.
package les

import (
	"io"
	"math"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	// After a connection has been ended or timed out, there is a waiting period
	// before it can be selected for connection again.
	// waiting period = base delay * (1 + random(1))
	// base delay = shortRetryDelay for the first shortRetryCnt times after a
	// successful connection, after that longRetryDelay is applied
	shortRetryCnt   = 5
	shortRetryDelay = time.Second * 5
	longRetryDelay  = time.Minute * 10
	// maxNewEntries is the maximum number of newly discovered (never connected) nodes.
	// If the limit is reached, the least recently discovered one is thrown out.
	maxNewEntries = 1000
	// maxKnownEntries is the maximum number of known (already connected) nodes.
	// If the limit is reached, the least recently connected one is thrown out.
	// (not that unlike new entries, known entries are persistent)
	maxKnownEntries = 1000
	// target for simultaneously connected servers
	targetServerCount = 5
	// target for servers selected from the known table
	// (we leave room for trying new ones if there is any)
	targetKnownSelect = 3
	// after dialTimeout, consider the server unavailable and adjust statistics
	dialTimeout = time.Second * 30
	// new entry selection weight calculation based on most recent discovery time:
	// unity until discoverExpireStart, then exponential decay with discoverExpireConst
	discoverExpireStart = time.Minute * 20
	discoverExpireConst = time.Minute * 20
	// known entry selection weight is dropped by a factor of exp(-failDropLn) after
	// each unsuccessful connection (restored after a successful one)
	failDropLn = 0.1
	// known node connection success and quality statistics have a long term average
	// and a short term value which is adjusted exponentially with a factor of
	// pstatRecentAdjust with each dial/connection and also returned exponentially
	// to the average with the time constant pstatReturnToMeanTC
	pstatRecentAdjust   = 0.1
	pstatReturnToMeanTC = time.Hour
	// node address selection weight is dropped by a factor of exp(-addrFailDropLn) after
	// each unsuccessful connection (restored after a successful one)
	addrFailDropLn = math.Ln2
)

// serverPool implements a pool for storing and selecting newly discovered and already
// known light server nodes. It received discovered nodes, stores statistics about
// known nodes and takes care of always having enough good quality servers connected.
type serverPool struct {
	db     ethdb.Database
	dbKey  []byte
	server *p2p.Server
	quit   chan struct{}
	wg     *sync.WaitGroup
	connWg sync.WaitGroup

	discSetPeriod chan time.Duration
	discNodes     chan *discv5.Node
	discLookups   chan bool

	entries              map[discover.NodeID]*poolEntry
	lock                 sync.Mutex
	timeout, enableRetry chan *poolEntry

	knownQueue, newQueue       poolEntryQueue
	knownSelect, newSelect     *weightedRandomSelect
	knownSelected, newSelected int
	fastDiscover               bool
}

// newServerPool creates a new serverPool instance
func newServerPool(db ethdb.Database, dbPrefix []byte, server *p2p.Server, topic discv5.Topic, quit chan struct{}, wg *sync.WaitGroup) *serverPool {
	pool := &serverPool{
		db:           db,
		dbKey:        append(dbPrefix, []byte(topic)...),
		server:       server,
		quit:         quit,
		wg:           wg,
		entries:      make(map[discover.NodeID]*poolEntry),
		timeout:      make(chan *poolEntry, 1),
		enableRetry:  make(chan *poolEntry, 1),
		knownSelect:  newWeightedRandomSelect(),
		newSelect:    newWeightedRandomSelect(),
		fastDiscover: true,
	}
	pool.knownQueue = newPoolEntryQueue(maxKnownEntries, pool.removeEntry)
	pool.newQueue = newPoolEntryQueue(maxNewEntries, pool.removeEntry)
	wg.Add(1)
	pool.loadNodes()
	pool.checkDial()

	if pool.server.DiscV5 != nil {
		pool.discSetPeriod = make(chan time.Duration, 1)
		pool.discNodes = make(chan *discv5.Node, 100)
		pool.discLookups = make(chan bool, 100)
		go pool.server.DiscV5.SearchTopic(topic, pool.discSetPeriod, pool.discNodes, pool.discLookups)
	}

	go pool.eventLoop()
	return pool
}

// connect should be called upon any incoming connection. If the connection has been
// dialed by the server pool recently, the appropriate pool entry is returned.
// Otherwise, the connection should be rejected.
// Note that whenever a connection has been accepted and a pool entry has been returned,
// disconnect should also always be called.
func (pool *serverPool) connect(id discover.NodeID, ip net.IP, port uint16) *poolEntry {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	entry := pool.entries[id]
	if entry == nil {
		return nil
	}
	glog.V(logger.Debug).Infof("connecting to %v, state: %v", id.String(), entry.state)
	if entry.state != psDialed {
		return nil
	}
	pool.connWg.Add(1)
	entry.state = psConnected
	addr := &poolEntryAddress{
		ip:       ip,
		port:     port,
		lastSeen: mclock.Now(),
	}
	entry.lastConnected = addr
	entry.addr = make(map[string]*poolEntryAddress)
	entry.addr[addr.strKey()] = addr
	entry.addrSelect = *newWeightedRandomSelect()
	entry.addrSelect.update(addr)
	return entry
}

// registered should be called after a successful handshake
func (pool *serverPool) registered(entry *poolEntry) {
	glog.V(logger.Debug).Infof("registered %v", entry.id.String())
	pool.lock.Lock()
	defer pool.lock.Unlock()

	entry.state = psRegistered
	if !entry.known {
		pool.newQueue.remove(entry)
		entry.known = true
	}
	pool.knownQueue.setLatest(entry)
	entry.shortRetry = shortRetryCnt
	entry.connectStats.add(1)
}

// disconnect should be called when ending a connection. Service quality statistics
// can be updated optionally (not updated if no registration happened, in this case
// only connection statistics are updated, just like in case of timeout)
func (pool *serverPool) disconnect(entry *poolEntry, quality float64, setQuality bool) {
	glog.V(logger.Debug).Infof("disconnected %v", entry.id.String())
	pool.lock.Lock()
	defer pool.lock.Unlock()

	if entry.state != psRegistered {
		setQuality = false
	}
	entry.state = psNotConnected
	if entry.knownSelected {
		pool.knownSelected--
	} else {
		pool.newSelected--
	}
	if setQuality {
		glog.V(logger.Debug).Infof("update quality %v %v", quality, entry.id.String())
		entry.qualityStats.add(quality)
	} else {
		glog.V(logger.Debug).Infof("do not update quality")
	}
	pool.setRetryDial(entry)
	pool.connWg.Done()
}

// eventLoop handles pool events and mutex locking for all internal functions
func (pool *serverPool) eventLoop() {
	lookupCnt := 0
	var convTime mclock.AbsTime
	pool.discSetPeriod <- time.Millisecond * 100
	for {
		select {
		case entry := <-pool.timeout:
			pool.lock.Lock()
			if !entry.removed {
				pool.checkDialTimeout(entry)
			}
			pool.lock.Unlock()

		case entry := <-pool.enableRetry:
			pool.lock.Lock()
			if !entry.removed {
				entry.delayedRetry = false
				pool.updateCheckDial(entry)
			}
			pool.lock.Unlock()

		case node := <-pool.discNodes:
			pool.lock.Lock()
			now := mclock.Now()
			id := discover.NodeID(node.ID)
			entry := pool.entries[id]
			if entry == nil {
				glog.V(logger.Debug).Infof("discovered %v", node.String())
				entry = &poolEntry{
					id:         id,
					addr:       make(map[string]*poolEntryAddress),
					addrSelect: *newWeightedRandomSelect(),
					shortRetry: shortRetryCnt,
				}
				pool.entries[id] = entry
			}
			entry.lastDiscovered = now
			addr := &poolEntryAddress{
				ip:   node.IP,
				port: node.TCP,
			}
			if a, ok := entry.addr[addr.strKey()]; ok {
				addr = a
			} else {
				entry.addr[addr.strKey()] = addr
			}
			addr.lastSeen = now
			entry.addrSelect.update(addr)
			if !entry.known {
				pool.newQueue.setLatest(entry)
			}
			pool.updateCheckDial(entry)
			pool.lock.Unlock()

		case conv := <-pool.discLookups:
			if conv {
				if lookupCnt == 0 {
					convTime = mclock.Now()
				}
				lookupCnt++
				if pool.fastDiscover && (lookupCnt == 50 || time.Duration(mclock.Now()-convTime) > time.Minute) {
					pool.fastDiscover = false
					pool.discSetPeriod <- time.Minute
				}
			}

		case <-pool.quit:
			close(pool.discSetPeriod)
			pool.connWg.Wait()
			pool.saveNodes()
			pool.wg.Done()
			return

		}
	}
}

// loadNodes loads known nodes and their statistics from the database
func (pool *serverPool) loadNodes() {
	enc, err := pool.db.Get(pool.dbKey)
	if err != nil {
		return
	}
	var list []*poolEntry
	err = rlp.DecodeBytes(enc, &list)
	if err != nil {
		glog.V(logger.Debug).Infof("node list decode error: %v", err)
		return
	}
	glog.V(logger.Debug).Infof("loaded node list")
	for _, e := range list {
		glog.V(logger.Debug).Infof(" adding node %v  fails: %v  connStats sum: %v cnt: %v  qualityStats sum: %v cnt: %v", e.id.String()+"@"+e.lastConnected.strKey(), e.lastConnected.fails, e.connectStats.sum, e.connectStats.cnt, e.qualityStats.sum, e.qualityStats.cnt)
		pool.entries[e.id] = e
		pool.knownQueue.setLatest(e)
		pool.knownSelect.update((*knownEntry)(e))
	}
}

// saveNodes saves known nodes and their statistics into the database. Nodes are
// ordered from least to most recently connected.
func (pool *serverPool) saveNodes() {
	list := make([]*poolEntry, len(pool.knownQueue.queue))
	for i, _ := range list {
		list[i] = pool.knownQueue.fetchOldest()
	}
	enc, err := rlp.EncodeToBytes(list)
	if err == nil {
		pool.db.Put(pool.dbKey, enc)
	}
}

// removeEntry removes a pool entry when the entry count limit is reached.
// Note that it is called by the new/known queues from which the entry has already
// been removed so removing it from the queues is not necessary.
func (pool *serverPool) removeEntry(entry *poolEntry) {
	pool.newSelect.remove((*discoveredEntry)(entry))
	pool.knownSelect.remove((*knownEntry)(entry))
	entry.removed = true
	delete(pool.entries, entry.id)
}

// setRetryDial starts the timer which will enable dialing a certain node again
func (pool *serverPool) setRetryDial(entry *poolEntry) {
	delay := longRetryDelay
	if entry.shortRetry > 0 {
		entry.shortRetry--
		delay = shortRetryDelay
	}
	delay += time.Duration(rand.Int63n(int64(delay) + 1))
	entry.delayedRetry = true
	go func() {
		select {
		case <-pool.quit:
		case <-time.After(delay):
			select {
			case <-pool.quit:
			case pool.enableRetry <- entry:
			}
		}
	}()
}

// updateCheckDial is called when an entry can potentially be dialed again. It updates
// its selection weights and checks if new dials can/should be made.
func (pool *serverPool) updateCheckDial(entry *poolEntry) {
	pool.newSelect.update((*discoveredEntry)(entry))
	pool.knownSelect.update((*knownEntry)(entry))
	pool.checkDial()
}

// checkDial checks if new dials can/should be made. It tries to select servers both
// based on good statistics and recent discovery.
func (pool *serverPool) checkDial() {
	fillWithKnownSelects := !pool.fastDiscover
	for pool.knownSelected < targetKnownSelect {
		entry := pool.knownSelect.choose()
		if entry == nil {
			fillWithKnownSelects = false
			break
		}
		pool.dial((*poolEntry)(entry.(*knownEntry)), true)
	}
	for pool.knownSelected+pool.newSelected < targetServerCount {
		entry := pool.newSelect.choose()
		if entry == nil {
			break
		}
		pool.dial((*poolEntry)(entry.(*discoveredEntry)), false)
	}
	if fillWithKnownSelects {
		// no more newly discovered nodes to select and since fast discover period
		// is over, we probably won't find more in the near future so select more
		// known entries if possible
		for pool.knownSelected < targetServerCount {
			entry := pool.knownSelect.choose()
			if entry == nil {
				break
			}
			pool.dial((*poolEntry)(entry.(*knownEntry)), true)
		}
	}
}

// dial initiates a new connection
func (pool *serverPool) dial(entry *poolEntry, knownSelected bool) {
	if entry.state != psNotConnected {
		return
	}
	entry.state = psDialed
	entry.knownSelected = knownSelected
	if knownSelected {
		pool.knownSelected++
	} else {
		pool.newSelected++
	}
	addr := entry.addrSelect.choose().(*poolEntryAddress)
	glog.V(logger.Debug).Infof("dialing %v out of %v, known: %v", entry.id.String()+"@"+addr.strKey(), len(entry.addr), knownSelected)
	entry.dialed = addr
	go func() {
		pool.server.AddPeer(discover.NewNode(entry.id, addr.ip, addr.port, addr.port))
		select {
		case <-pool.quit:
		case <-time.After(dialTimeout):
			select {
			case <-pool.quit:
			case pool.timeout <- entry:
			}
		}
	}()
}

// checkDialTimeout checks if the node is still in dialed state and if so, resets it
// and adjusts connection statistics accordingly.
func (pool *serverPool) checkDialTimeout(entry *poolEntry) {
	if entry.state != psDialed {
		return
	}
	glog.V(logger.Debug).Infof("timeout %v", entry.id.String()+"@"+entry.dialed.strKey())
	entry.state = psNotConnected
	if entry.knownSelected {
		pool.knownSelected--
	} else {
		pool.newSelected--
	}
	entry.connectStats.add(0)
	entry.dialed.fails++
	pool.setRetryDial(entry)
}

const (
	psNotConnected = iota
	psDialed
	psConnected
	psRegistered
)

// poolEntry represents a server node and stores its current state and statistics.
type poolEntry struct {
	id                    discover.NodeID
	addr                  map[string]*poolEntryAddress
	lastConnected, dialed *poolEntryAddress
	addrSelect            weightedRandomSelect

	lastDiscovered             mclock.AbsTime
	known, knownSelected       bool
	connectStats, qualityStats poolStats
	state                      int
	queueIdx                   int
	removed                    bool

	delayedRetry bool
	shortRetry   int
}

func (e *poolEntry) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{e.id, e.lastConnected.ip, e.lastConnected.port, e.lastConnected.fails, &e.connectStats, &e.qualityStats})
}

func (e *poolEntry) DecodeRLP(s *rlp.Stream) error {
	var entry struct {
		ID           discover.NodeID
		IP           net.IP
		Port         uint16
		Fails        uint
		CStat, QStat poolStats
	}
	if err := s.Decode(&entry); err != nil {
		return err
	}
	addr := &poolEntryAddress{ip: entry.IP, port: entry.Port, fails: entry.Fails, lastSeen: mclock.Now()}
	e.id = entry.ID
	e.addr = make(map[string]*poolEntryAddress)
	e.addr[addr.strKey()] = addr
	e.addrSelect = *newWeightedRandomSelect()
	e.addrSelect.update(addr)
	e.lastConnected = addr
	e.connectStats = entry.CStat
	e.qualityStats = entry.QStat
	e.shortRetry = shortRetryCnt
	e.known = true
	return nil
}

// discoveredEntry implements wrsItem
type discoveredEntry poolEntry

// Weight calculates random selection weight for newly discovered entries
func (e *discoveredEntry) Weight() int64 {
	if e.state != psNotConnected || e.delayedRetry {
		return 0
	}
	t := time.Duration(mclock.Now() - e.lastDiscovered)
	if t <= discoverExpireStart {
		return 1000000000
	} else {
		return int64(1000000000 * math.Exp(-float64(t-discoverExpireStart)/float64(discoverExpireConst)))
	}
}

// knownEntry implements wrsItem
type knownEntry poolEntry

// Weight calculates random selection weight for known entries
func (e *knownEntry) Weight() int64 {
	if e.state != psNotConnected || !e.known || e.delayedRetry {
		return 0
	}
	return int64(1000000000 * e.connectStats.recentAvg() * (e.qualityStats.recentAvg() + 0.001) * math.Exp(-float64(e.lastConnected.fails)*failDropLn))
}

// poolEntryAddress is a separate object because currently it is necessary to remember
// multiple potential network addresses for a pool entry. This will be removed after
// the final implementation of v5 discovery which will retrieve signed and serial
// numbered advertisements, making it clear which IP/port is the latest one.
type poolEntryAddress struct {
	ip       net.IP
	port     uint16
	lastSeen mclock.AbsTime // last time it was discovered, connected or loaded from db
	fails    uint           // connection failures since last successful connection (persistent)
}

func (a *poolEntryAddress) Weight() int64 {
	t := time.Duration(mclock.Now() - a.lastSeen)
	return int64(1000000*math.Exp(-float64(t)/float64(discoverExpireConst)-float64(a.fails)*addrFailDropLn)) + 1
}

func (a *poolEntryAddress) strKey() string {
	return a.ip.String() + ":" + strconv.Itoa(int(a.port))
}

// poolStats implement statistics for a certain quantity with a long term average
// and a short term value which is adjusted exponentially with a factor of
// pstatRecentAdjust with each update and also returned exponentially to the
// average with the time constant pstatReturnToMeanTC
type poolStats struct {
	sum, avg, recent float64
	cnt              uint
	lastRecalc       mclock.AbsTime
}

// init initializes stats with a long term sum/update count pair retrieved from the database
func (s *poolStats) init(sum float64, cnt uint) {
	s.sum = sum
	s.cnt = cnt
	var avg float64
	if cnt > 0 {
		avg = s.sum / float64(cnt)
	}
	s.avg = avg
	s.recent = avg
	s.lastRecalc = mclock.Now()
}

// recalc recalculates recent value return-to-mean and long term average
func (s *poolStats) recalc() {
	now := mclock.Now()
	s.recent = s.avg + (s.recent-s.avg)*math.Exp(-float64(now-s.lastRecalc)/float64(pstatReturnToMeanTC))
	if s.cnt > 0 {
		s.avg = s.sum / float64(s.cnt)
	}
	s.lastRecalc = now
}

// add updates the stats with a new value
func (s *poolStats) add(val float64) {
	s.cnt++
	s.sum += val
	s.recalc()
}

// recentAvg returns the short-term adjusted average
func (s *poolStats) recentAvg() float64 {
	s.recalc()
	return s.recent
}

func (s *poolStats) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{math.Float64bits(s.sum), s.cnt})
}

func (s *poolStats) DecodeRLP(st *rlp.Stream) error {
	var stats struct {
		SumUint uint64
		Cnt     uint
	}
	if err := st.Decode(&stats); err != nil {
		return err
	}
	s.init(math.Float64frombits(stats.SumUint), stats.Cnt)
	return nil
}

// poolEntryQueue keeps track of its least recently accessed entries and removes
// them when the number of entries reaches the limit
type poolEntryQueue struct {
	queue                  map[int]*poolEntry // known nodes indexed by their latest lastConnCnt value
	newPtr, oldPtr, maxCnt int
	removeFromPool         func(*poolEntry)
}

// newPoolEntryQueue returns a new poolEntryQueue
func newPoolEntryQueue(maxCnt int, removeFromPool func(*poolEntry)) poolEntryQueue {
	return poolEntryQueue{queue: make(map[int]*poolEntry), maxCnt: maxCnt, removeFromPool: removeFromPool}
}

// fetchOldest returns and removes the least recently accessed entry
func (q *poolEntryQueue) fetchOldest() *poolEntry {
	if len(q.queue) == 0 {
		return nil
	}
	for {
		if e := q.queue[q.oldPtr]; e != nil {
			delete(q.queue, q.oldPtr)
			q.oldPtr++
			return e
		}
		q.oldPtr++
	}
}

// remove removes an entry from the queue
func (q *poolEntryQueue) remove(entry *poolEntry) {
	if q.queue[entry.queueIdx] == entry {
		delete(q.queue, entry.queueIdx)
	}
}

// setLatest adds or updates a recently accessed entry. It also checks if an old entry
// needs to be removed and removes it from the parent pool too with a callback function.
func (q *poolEntryQueue) setLatest(entry *poolEntry) {
	if q.queue[entry.queueIdx] == entry {
		delete(q.queue, entry.queueIdx)
	} else {
		if len(q.queue) == q.maxCnt {
			e := q.fetchOldest()
			q.remove(e)
			q.removeFromPool(e)
		}
	}
	entry.queueIdx = q.newPtr
	q.queue[entry.queueIdx] = entry
	q.newPtr++
}
