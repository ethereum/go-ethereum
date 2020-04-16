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
	"crypto/ecdsa"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/les/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/enode"
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
	// targetConnTime is the minimum expected connection duration before a server
	// drops a client without any specific reason
	targetConnTime = time.Minute * 10
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
	pstatReturnToMeanTC = time.Hour
	// node address selection weight is dropped by a factor of exp(-addrFailDropLn) after
	// each unsuccessful connection (restored after a successful one)
	addrFailDropLn = math.Ln2
	// responseScoreTC and delayScoreTC are exponential decay time constants for
	// calculating selection chances from response times and block delay times
	responseScoreTC = time.Millisecond * 100
	delayScoreTC    = time.Second * 5
	timeoutPow      = 10
	// initStatsWeight is used to initialize previously unknown peers with good
	// statistics to give a chance to prove themselves
	initStatsWeight = 1
)

// connReq represents a request for peer connection.
type connReq struct {
	p      *serverPeer
	node   *enode.Node
	result chan *poolEntry
}

// disconnReq represents a request for peer disconnection.
type disconnReq struct {
	entry   *poolEntry
	stopped bool
	done    chan struct{}
}

// registerReq represents a request for peer registration.
type registerReq struct {
	entry *poolEntry
	done  chan struct{}
}

// serverPool implements a pool for storing and selecting newly discovered and already
// known light server nodes. It received discovered nodes, stores statistics about
// known nodes and takes care of always having enough good quality servers connected.
type serverPool struct {
	db     ethdb.Database
	dbKey  []byte
	server *p2p.Server
	connWg sync.WaitGroup

	topic discv5.Topic

	discSetPeriod chan time.Duration
	discNodes     chan *enode.Node
	discLookups   chan bool

	trustedNodes         map[enode.ID]*enode.Node
	entries              map[enode.ID]*poolEntry
	timeout, enableRetry chan *poolEntry
	adjustStats          chan poolStatAdjust

	knownQueue, newQueue       poolEntryQueue
	knownSelect, newSelect     *utils.WeightedRandomSelect
	knownSelected, newSelected int
	fastDiscover               bool
	connCh                     chan *connReq
	disconnCh                  chan *disconnReq
	registerCh                 chan *registerReq

	closeCh chan struct{}
	wg      sync.WaitGroup
}

// newServerPool creates a new serverPool instance
func newServerPool(db ethdb.Database, ulcServers []string) *serverPool {
	pool := &serverPool{
		db:           db,
		entries:      make(map[enode.ID]*poolEntry),
		timeout:      make(chan *poolEntry, 1),
		adjustStats:  make(chan poolStatAdjust, 100),
		enableRetry:  make(chan *poolEntry, 1),
		connCh:       make(chan *connReq),
		disconnCh:    make(chan *disconnReq),
		registerCh:   make(chan *registerReq),
		closeCh:      make(chan struct{}),
		knownSelect:  utils.NewWeightedRandomSelect(),
		newSelect:    utils.NewWeightedRandomSelect(),
		fastDiscover: true,
		trustedNodes: parseTrustedNodes(ulcServers),
	}

	pool.knownQueue = newPoolEntryQueue(maxKnownEntries, pool.removeEntry)
	pool.newQueue = newPoolEntryQueue(maxNewEntries, pool.removeEntry)
	return pool
}

func (pool *serverPool) start(server *p2p.Server, topic discv5.Topic) {
	pool.server = server
	pool.topic = topic
	pool.dbKey = append([]byte("serverPool/"), []byte(topic)...)
	pool.loadNodes()
	pool.connectToTrustedNodes()

	if pool.server.DiscV5 != nil {
		pool.discSetPeriod = make(chan time.Duration, 1)
		pool.discNodes = make(chan *enode.Node, 100)
		pool.discLookups = make(chan bool, 100)
		go pool.discoverNodes()
	}
	pool.checkDial()
	pool.wg.Add(1)
	go pool.eventLoop()

	// Inject the bootstrap nodes as initial dial candiates.
	pool.wg.Add(1)
	go func() {
		defer pool.wg.Done()
		for _, n := range server.BootstrapNodes {
			select {
			case pool.discNodes <- n:
			case <-pool.closeCh:
				return
			}
		}
	}()
}

func (pool *serverPool) stop() {
	close(pool.closeCh)
	pool.wg.Wait()
}

// discoverNodes wraps SearchTopic, converting result nodes to enode.Node.
func (pool *serverPool) discoverNodes() {
	ch := make(chan *discv5.Node)
	go func() {
		pool.server.DiscV5.SearchTopic(pool.topic, pool.discSetPeriod, ch, pool.discLookups)
		close(ch)
	}()
	for n := range ch {
		pubkey, err := decodePubkey64(n.ID[:])
		if err != nil {
			continue
		}
		pool.discNodes <- enode.NewV4(pubkey, n.IP, int(n.TCP), int(n.UDP))
	}
}

// connect should be called upon any incoming connection. If the connection has been
// dialed by the server pool recently, the appropriate pool entry is returned.
// Otherwise, the connection should be rejected.
// Note that whenever a connection has been accepted and a pool entry has been returned,
// disconnect should also always be called.
func (pool *serverPool) connect(p *serverPeer, node *enode.Node) *poolEntry {
	log.Debug("Connect new entry", "enode", p.id)
	req := &connReq{p: p, node: node, result: make(chan *poolEntry, 1)}
	select {
	case pool.connCh <- req:
	case <-pool.closeCh:
		return nil
	}
	return <-req.result
}

// registered should be called after a successful handshake
func (pool *serverPool) registered(entry *poolEntry) {
	log.Debug("Registered new entry", "enode", entry.node.ID())
	req := &registerReq{entry: entry, done: make(chan struct{})}
	select {
	case pool.registerCh <- req:
	case <-pool.closeCh:
		return
	}
	<-req.done
}

// disconnect should be called when ending a connection. Service quality statistics
// can be updated optionally (not updated if no registration happened, in this case
// only connection statistics are updated, just like in case of timeout)
func (pool *serverPool) disconnect(entry *poolEntry) {
	stopped := false
	select {
	case <-pool.closeCh:
		stopped = true
	default:
	}
	log.Debug("Disconnected old entry", "enode", entry.node.ID())
	req := &disconnReq{entry: entry, stopped: stopped, done: make(chan struct{})}

	// Block until disconnection request is served.
	pool.disconnCh <- req
	<-req.done
}

const (
	pseBlockDelay = iota
	pseResponseTime
	pseResponseTimeout
)

// poolStatAdjust records are sent to adjust peer block delay/response time statistics
type poolStatAdjust struct {
	adjustType int
	entry      *poolEntry
	time       time.Duration
}

// adjustBlockDelay adjusts the block announce delay statistics of a node
func (pool *serverPool) adjustBlockDelay(entry *poolEntry, time time.Duration) {
	if entry == nil {
		return
	}
	pool.adjustStats <- poolStatAdjust{pseBlockDelay, entry, time}
}

// adjustResponseTime adjusts the request response time statistics of a node
func (pool *serverPool) adjustResponseTime(entry *poolEntry, time time.Duration, timeout bool) {
	if entry == nil {
		return
	}
	if timeout {
		pool.adjustStats <- poolStatAdjust{pseResponseTimeout, entry, time}
	} else {
		pool.adjustStats <- poolStatAdjust{pseResponseTime, entry, time}
	}
}

// eventLoop handles pool events and mutex locking for all internal functions
func (pool *serverPool) eventLoop() {
	defer pool.wg.Done()
	lookupCnt := 0
	var convTime mclock.AbsTime
	if pool.discSetPeriod != nil {
		pool.discSetPeriod <- time.Millisecond * 100
	}

	// disconnect updates service quality statistics depending on the connection time
	// and disconnection initiator.
	disconnect := func(req *disconnReq, stopped bool) {
		// Handle peer disconnection requests.
		entry := req.entry
		if entry.state == psRegistered {
			connAdjust := float64(mclock.Now()-entry.regTime) / float64(targetConnTime)
			if connAdjust > 1 {
				connAdjust = 1
			}
			if stopped {
				// disconnect requested by ourselves.
				entry.connectStats.add(1, connAdjust)
			} else {
				// disconnect requested by server side.
				entry.connectStats.add(connAdjust, 1)
			}
		}
		entry.state = psNotConnected

		if entry.knownSelected {
			pool.knownSelected--
		} else {
			pool.newSelected--
		}
		pool.setRetryDial(entry)
		pool.connWg.Done()
		close(req.done)
	}

	for {
		select {
		case entry := <-pool.timeout:
			if !entry.removed {
				pool.checkDialTimeout(entry)
			}

		case entry := <-pool.enableRetry:
			if !entry.removed {
				entry.delayedRetry = false
				pool.updateCheckDial(entry)
			}

		case adj := <-pool.adjustStats:
			switch adj.adjustType {
			case pseBlockDelay:
				adj.entry.delayStats.add(float64(adj.time), 1)
			case pseResponseTime:
				adj.entry.responseStats.add(float64(adj.time), 1)
				adj.entry.timeoutStats.add(0, 1)
			case pseResponseTimeout:
				adj.entry.timeoutStats.add(1, 1)
			}

		case node := <-pool.discNodes:
			if pool.trustedNodes[node.ID()] == nil {
				entry := pool.findOrNewNode(node)
				pool.updateCheckDial(entry)
			}

		case conv := <-pool.discLookups:
			if conv {
				if lookupCnt == 0 {
					convTime = mclock.Now()
				}
				lookupCnt++
				if pool.fastDiscover && (lookupCnt == 50 || time.Duration(mclock.Now()-convTime) > time.Minute) {
					pool.fastDiscover = false
					if pool.discSetPeriod != nil {
						pool.discSetPeriod <- time.Minute
					}
				}
			}

		case req := <-pool.connCh:
			if pool.trustedNodes[req.p.ID()] != nil {
				// ignore trusted nodes
				req.result <- &poolEntry{trusted: true}
			} else {
				// Handle peer connection requests.
				entry := pool.entries[req.p.ID()]
				if entry == nil {
					entry = pool.findOrNewNode(req.node)
				}
				if entry.state == psConnected || entry.state == psRegistered {
					req.result <- nil
					continue
				}
				pool.connWg.Add(1)
				entry.peer = req.p
				entry.state = psConnected
				addr := &poolEntryAddress{
					ip:       req.node.IP(),
					port:     uint16(req.node.TCP()),
					lastSeen: mclock.Now(),
				}
				entry.lastConnected = addr
				entry.addr = make(map[string]*poolEntryAddress)
				entry.addr[addr.strKey()] = addr
				entry.addrSelect = *utils.NewWeightedRandomSelect()
				entry.addrSelect.Update(addr)
				req.result <- entry
			}

		case req := <-pool.registerCh:
			if req.entry.trusted {
				continue
			}
			// Handle peer registration requests.
			entry := req.entry
			entry.state = psRegistered
			entry.regTime = mclock.Now()
			if !entry.known {
				pool.newQueue.remove(entry)
				entry.known = true
			}
			pool.knownQueue.setLatest(entry)
			entry.shortRetry = shortRetryCnt
			close(req.done)

		case req := <-pool.disconnCh:
			if req.entry.trusted {
				continue
			}
			// Handle peer disconnection requests.
			disconnect(req, req.stopped)

		case <-pool.closeCh:
			if pool.discSetPeriod != nil {
				close(pool.discSetPeriod)
			}

			// Spawn a goroutine to close the disconnCh after all connections are disconnected.
			go func() {
				pool.connWg.Wait()
				close(pool.disconnCh)
			}()

			// Handle all remaining disconnection requests before exit.
			for req := range pool.disconnCh {
				disconnect(req, true)
			}
			pool.saveNodes()
			return
		}
	}
}

func (pool *serverPool) findOrNewNode(node *enode.Node) *poolEntry {
	now := mclock.Now()
	entry := pool.entries[node.ID()]
	if entry == nil {
		log.Debug("Discovered new entry", "id", node.ID())
		entry = &poolEntry{
			node:       node,
			addr:       make(map[string]*poolEntryAddress),
			addrSelect: *utils.NewWeightedRandomSelect(),
			shortRetry: shortRetryCnt,
		}
		pool.entries[node.ID()] = entry
		// initialize previously unknown peers with good statistics to give a chance to prove themselves
		entry.connectStats.add(1, initStatsWeight)
		entry.delayStats.add(0, initStatsWeight)
		entry.responseStats.add(0, initStatsWeight)
		entry.timeoutStats.add(0, initStatsWeight)
	}
	entry.lastDiscovered = now
	addr := &poolEntryAddress{ip: node.IP(), port: uint16(node.TCP())}
	if a, ok := entry.addr[addr.strKey()]; ok {
		addr = a
	} else {
		entry.addr[addr.strKey()] = addr
	}
	addr.lastSeen = now
	entry.addrSelect.Update(addr)
	if !entry.known {
		pool.newQueue.setLatest(entry)
	}
	return entry
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
		log.Debug("Failed to decode node list", "err", err)
		return
	}
	for _, e := range list {
		log.Debug("Loaded server stats", "id", e.node.ID(), "fails", e.lastConnected.fails,
			"conn", fmt.Sprintf("%v/%v", e.connectStats.avg, e.connectStats.weight),
			"delay", fmt.Sprintf("%v/%v", time.Duration(e.delayStats.avg), e.delayStats.weight),
			"response", fmt.Sprintf("%v/%v", time.Duration(e.responseStats.avg), e.responseStats.weight),
			"timeout", fmt.Sprintf("%v/%v", e.timeoutStats.avg, e.timeoutStats.weight))
		pool.entries[e.node.ID()] = e
		if pool.trustedNodes[e.node.ID()] == nil {
			pool.knownQueue.setLatest(e)
			pool.knownSelect.Update((*knownEntry)(e))
		}
	}
}

// connectToTrustedNodes adds trusted server nodes as static trusted peers.
//
// Note: trusted nodes are not handled by the server pool logic, they are not
// added to either the known or new selection pools. They are connected/reconnected
// by p2p.Server whenever possible.
func (pool *serverPool) connectToTrustedNodes() {
	//connect to trusted nodes
	for _, node := range pool.trustedNodes {
		pool.server.AddTrustedPeer(node)
		pool.server.AddPeer(node)
		log.Debug("Added trusted node", "id", node.ID().String())
	}
}

// parseTrustedNodes returns valid and parsed enodes
func parseTrustedNodes(trustedNodes []string) map[enode.ID]*enode.Node {
	nodes := make(map[enode.ID]*enode.Node)

	for _, node := range trustedNodes {
		node, err := enode.Parse(enode.ValidSchemes, node)
		if err != nil {
			log.Warn("Trusted node URL invalid", "enode", node, "err", err)
			continue
		}
		nodes[node.ID()] = node
	}
	return nodes
}

// saveNodes saves known nodes and their statistics into the database. Nodes are
// ordered from least to most recently connected.
func (pool *serverPool) saveNodes() {
	list := make([]*poolEntry, len(pool.knownQueue.queue))
	for i := range list {
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
	pool.newSelect.Remove((*discoveredEntry)(entry))
	pool.knownSelect.Remove((*knownEntry)(entry))
	entry.removed = true
	delete(pool.entries, entry.node.ID())
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
		case <-pool.closeCh:
		case <-time.After(delay):
			select {
			case <-pool.closeCh:
			case pool.enableRetry <- entry:
			}
		}
	}()
}

// updateCheckDial is called when an entry can potentially be dialed again. It updates
// its selection weights and checks if new dials can/should be made.
func (pool *serverPool) updateCheckDial(entry *poolEntry) {
	pool.newSelect.Update((*discoveredEntry)(entry))
	pool.knownSelect.Update((*knownEntry)(entry))
	pool.checkDial()
}

// checkDial checks if new dials can/should be made. It tries to select servers both
// based on good statistics and recent discovery.
func (pool *serverPool) checkDial() {
	fillWithKnownSelects := !pool.fastDiscover
	for pool.knownSelected < targetKnownSelect {
		entry := pool.knownSelect.Choose()
		if entry == nil {
			fillWithKnownSelects = false
			break
		}
		pool.dial((*poolEntry)(entry.(*knownEntry)), true)
	}
	for pool.knownSelected+pool.newSelected < targetServerCount {
		entry := pool.newSelect.Choose()
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
			entry := pool.knownSelect.Choose()
			if entry == nil {
				break
			}
			pool.dial((*poolEntry)(entry.(*knownEntry)), true)
		}
	}
}

// dial initiates a new connection
func (pool *serverPool) dial(entry *poolEntry, knownSelected bool) {
	if pool.server == nil || entry.state != psNotConnected {
		return
	}
	entry.state = psDialed
	entry.knownSelected = knownSelected
	if knownSelected {
		pool.knownSelected++
	} else {
		pool.newSelected++
	}
	addr := entry.addrSelect.Choose().(*poolEntryAddress)
	log.Debug("Dialing new peer", "lesaddr", entry.node.ID().String()+"@"+addr.strKey(), "set", len(entry.addr), "known", knownSelected)
	entry.dialed = addr
	go func() {
		pool.server.AddPeer(entry.node)
		select {
		case <-pool.closeCh:
		case <-time.After(dialTimeout):
			select {
			case <-pool.closeCh:
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
	log.Debug("Dial timeout", "lesaddr", entry.node.ID().String()+"@"+entry.dialed.strKey())
	entry.state = psNotConnected
	if entry.knownSelected {
		pool.knownSelected--
	} else {
		pool.newSelected--
	}
	entry.connectStats.add(0, 1)
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
	peer                  *serverPeer
	pubkey                [64]byte // secp256k1 key of the node
	addr                  map[string]*poolEntryAddress
	node                  *enode.Node
	lastConnected, dialed *poolEntryAddress
	addrSelect            utils.WeightedRandomSelect

	lastDiscovered                mclock.AbsTime
	known, knownSelected, trusted bool
	connectStats, delayStats      poolStats
	responseStats, timeoutStats   poolStats
	state                         int
	regTime                       mclock.AbsTime
	queueIdx                      int
	removed                       bool

	delayedRetry bool
	shortRetry   int
}

// poolEntryEnc is the RLP encoding of poolEntry.
type poolEntryEnc struct {
	Pubkey                     []byte
	IP                         net.IP
	Port                       uint16
	Fails                      uint
	CStat, DStat, RStat, TStat poolStats
}

func (e *poolEntry) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &poolEntryEnc{
		Pubkey: encodePubkey64(e.node.Pubkey()),
		IP:     e.lastConnected.ip,
		Port:   e.lastConnected.port,
		Fails:  e.lastConnected.fails,
		CStat:  e.connectStats,
		DStat:  e.delayStats,
		RStat:  e.responseStats,
		TStat:  e.timeoutStats,
	})
}

func (e *poolEntry) DecodeRLP(s *rlp.Stream) error {
	var entry poolEntryEnc
	if err := s.Decode(&entry); err != nil {
		return err
	}
	pubkey, err := decodePubkey64(entry.Pubkey)
	if err != nil {
		return err
	}
	addr := &poolEntryAddress{ip: entry.IP, port: entry.Port, fails: entry.Fails, lastSeen: mclock.Now()}
	e.node = enode.NewV4(pubkey, entry.IP, int(entry.Port), int(entry.Port))
	e.addr = make(map[string]*poolEntryAddress)
	e.addr[addr.strKey()] = addr
	e.addrSelect = *utils.NewWeightedRandomSelect()
	e.addrSelect.Update(addr)
	e.lastConnected = addr
	e.connectStats = entry.CStat
	e.delayStats = entry.DStat
	e.responseStats = entry.RStat
	e.timeoutStats = entry.TStat
	e.shortRetry = shortRetryCnt
	e.known = true
	return nil
}

func encodePubkey64(pub *ecdsa.PublicKey) []byte {
	return crypto.FromECDSAPub(pub)[1:]
}

func decodePubkey64(b []byte) (*ecdsa.PublicKey, error) {
	return crypto.UnmarshalPubkey(append([]byte{0x04}, b...))
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
	}
	return int64(1000000000 * math.Exp(-float64(t-discoverExpireStart)/float64(discoverExpireConst)))
}

// knownEntry implements wrsItem
type knownEntry poolEntry

// Weight calculates random selection weight for known entries
func (e *knownEntry) Weight() int64 {
	if e.state != psNotConnected || !e.known || e.delayedRetry {
		return 0
	}
	return int64(1000000000 * e.connectStats.recentAvg() * math.Exp(-float64(e.lastConnected.fails)*failDropLn-e.responseStats.recentAvg()/float64(responseScoreTC)-e.delayStats.recentAvg()/float64(delayScoreTC)) * math.Pow(1-e.timeoutStats.recentAvg(), timeoutPow))
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
	sum, weight, avg, recent float64
	lastRecalc               mclock.AbsTime
}

// init initializes stats with a long term sum/update count pair retrieved from the database
func (s *poolStats) init(sum, weight float64) {
	s.sum = sum
	s.weight = weight
	var avg float64
	if weight > 0 {
		avg = s.sum / weight
	}
	s.avg = avg
	s.recent = avg
	s.lastRecalc = mclock.Now()
}

// recalc recalculates recent value return-to-mean and long term average
func (s *poolStats) recalc() {
	now := mclock.Now()
	s.recent = s.avg + (s.recent-s.avg)*math.Exp(-float64(now-s.lastRecalc)/float64(pstatReturnToMeanTC))
	if s.sum == 0 {
		s.avg = 0
	} else {
		if s.sum > s.weight*1e30 {
			s.avg = 1e30
		} else {
			s.avg = s.sum / s.weight
		}
	}
	s.lastRecalc = now
}

// add updates the stats with a new value
func (s *poolStats) add(value, weight float64) {
	s.weight += weight
	s.sum += value * weight
	s.recalc()
}

// recentAvg returns the short-term adjusted average
func (s *poolStats) recentAvg() float64 {
	s.recalc()
	return s.recent
}

func (s *poolStats) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{math.Float64bits(s.sum), math.Float64bits(s.weight)})
}

func (s *poolStats) DecodeRLP(st *rlp.Stream) error {
	var stats struct {
		SumUint, WeightUint uint64
	}
	if err := st.Decode(&stats); err != nil {
		return err
	}
	s.init(math.Float64frombits(stats.SumUint), math.Float64frombits(stats.WeightUint))
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
