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

package discv5

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	errInvalidEvent = errors.New("invalid in current state")
	errNoQuery      = errors.New("no pending query")
	errWrongAddress = errors.New("unknown sender address")
)

const (
	autoRefreshInterval = 1 * time.Hour
	seedCount           = 30
	seedMaxAge          = 5 * 24 * time.Hour
)

const testTopic = "foo"

const (
	printDebugLogs   = false
	printTestImgLogs = false
)

func debugLog(s string) {
	if printDebugLogs {
		fmt.Println(s)
	}
}

// BootNodes are the enode URLs of the P2P bootstrap nodes for the experimental RLPx v5 "Topic Discovery" network
// warning: local bootnodes for testing!!!
var BootNodes = []*Node{
	//MustParseNode("enode://6f974ede10d07334e7e651c1501cb540d087dd3a6dea81432620895c913f281790b49459d72cb8011bfbbfbd24fad956356189c31b7181a96cd44ccfb68bfc71@127.0.0.1:30301"),
	MustParseNode("enode://0cc5f5ffb5d9098c8b8c62325f3797f56509bff942704687b6530992ac706e2cb946b90a34f1f19548cd3c7baccbcaea354531e5983c7d1bc0dee16ce4b6440b@40.118.3.223:30305"),
}

// Network manages the table and all protocol interaction.
type Network struct {
	db   *nodeDB // database of known nodes
	conn transport

	closed           chan struct{}          // closed when loop is done
	closeReq         chan struct{}          // 'request to close'
	refreshReq       chan []*Node           // lookups ask for refresh on this channel
	refreshResp      chan (<-chan struct{}) // ...and get the channel to block on from this one
	read             chan ingressPacket     // ingress packets arrive here
	timeout          chan timeoutEvent
	queryReq         chan *findnodeQuery // lookups submit findnode queries on this channel
	tableOpReq       chan func()
	tableOpResp      chan struct{}
	topicRegisterReq chan topicRegisterReq
	topicSearchReq   chan topicSearchReq
	bucketFillChn    chan chan struct{}

	// State of the main loop.
	tab           *Table
	topictab      *topicTable
	ticketStore   *ticketStore
	nursery       []*Node
	nodes         map[NodeID]*Node // tracks active nodes with state != known
	timeoutTimers map[timeoutEvent]*time.Timer

	// Revalidation queues.
	// Nodes put on these queues will be pinged eventually.
	slowRevalidateQueue []*Node
	fastRevalidateQueue []*Node

	// Buffers for state transition.
	sendBuf []*ingressPacket
}

// transport is implemented by the UDP transport.
// it is an interface so we can test without opening lots of UDP
// sockets and without generating a private key.
type transport interface {
	sendPing(remote *Node, remoteAddr *net.UDPAddr, topics []Topic) (hash []byte)
	sendNeighbours(remote *Node, nodes []*Node)
	sendFindnodeHash(remote *Node, target common.Hash)
	sendTopicRegister(remote *Node, topics []Topic, topicIdx int, pong []byte)
	sendTopicNodes(remote *Node, queryHash common.Hash, nodes []*Node)

	send(remote *Node, ptype nodeEvent, p interface{}) (hash []byte)

	localAddr() *net.UDPAddr
	Close()
}

type findnodeQuery struct {
	remote   *Node
	target   common.Hash
	reply    chan<- []*Node
	nresults int // counter for received nodes
}

type topicRegisterReq struct {
	add   bool
	topic Topic
}

type topicSearchReq struct {
	topic Topic
	found chan<- string
}

type timeoutEvent struct {
	ev   nodeEvent
	node *Node
}

func newNetwork(conn transport, ourPubkey ecdsa.PublicKey, natm nat.Interface, dbPath string) (*Network, error) {
	ourID := PubkeyID(&ourPubkey)

	var db *nodeDB
	if dbPath != "<no database>" {
		var err error
		if db, err = newNodeDB(dbPath, Version, ourID); err != nil {
			return nil, err
		}
	}

	tab := newTable(ourID, conn.localAddr())
	net := &Network{
		db:               db,
		conn:             conn,
		tab:              tab,
		topictab:         newTopicTable(db, tab.self),
		ticketStore:      newTicketStore(),
		refreshReq:       make(chan []*Node),
		refreshResp:      make(chan (<-chan struct{})),
		closed:           make(chan struct{}),
		closeReq:         make(chan struct{}),
		read:             make(chan ingressPacket, 100),
		timeout:          make(chan timeoutEvent),
		timeoutTimers:    make(map[timeoutEvent]*time.Timer),
		tableOpReq:       make(chan func()),
		tableOpResp:      make(chan struct{}),
		queryReq:         make(chan *findnodeQuery),
		topicRegisterReq: make(chan topicRegisterReq),
		topicSearchReq:   make(chan topicSearchReq),
		bucketFillChn:    make(chan chan struct{}, 1),
		nodes:            make(map[NodeID]*Node),
	}
	go net.loop()
	return net, nil
}

// Close terminates the network listener and flushes the node database.
func (net *Network) Close() {
	net.conn.Close()
	select {
	case <-net.closed:
	case net.closeReq <- struct{}{}:
		<-net.closed
	}
}

// Self returns the local node.
// The returned node should not be modified by the caller.
func (net *Network) Self() *Node {
	return net.tab.self
}

// ReadRandomNodes fills the given slice with random nodes from the
// table. It will not write the same node more than once. The nodes in
// the slice are copies and can be modified by the caller.
func (net *Network) ReadRandomNodes(buf []*Node) (n int) {
	net.reqTableOp(func() { n = net.tab.readRandomNodes(buf) })
	return n
}

// SetFallbackNodes sets the initial points of contact. These nodes
// are used to connect to the network if the table is empty and there
// are no known nodes in the database.
func (net *Network) SetFallbackNodes(nodes []*Node) error {
	nursery := make([]*Node, 0, len(nodes))
	for _, n := range nodes {
		if err := n.validateComplete(); err != nil {
			return fmt.Errorf("bad bootstrap/fallback node %q (%v)", n, err)
		}
		// Recompute cpy.sha because the node might not have been
		// created by NewNode or ParseNode.
		cpy := *n
		cpy.sha = crypto.Keccak256Hash(n.ID[:])
		nursery = append(nursery, &cpy)
	}
	net.reqRefresh(nursery)
	return nil
}

// Resolve searches for a specific node with the given ID.
// It returns nil if the node could not be found.
func (net *Network) Resolve(targetID NodeID) *Node {
	result := net.lookup(crypto.Keccak256Hash(targetID[:]), true)
	for _, n := range result {
		if n.ID == targetID {
			return n
		}
	}
	return nil
}

// Lookup performs a network search for nodes close
// to the given target. It approaches the target by querying
// nodes that are closer to it on each iteration.
// The given target does not need to be an actual node
// identifier.
//
// The local node may be included in the result.
func (net *Network) Lookup(targetID NodeID) []*Node {
	return net.lookup(crypto.Keccak256Hash(targetID[:]), false)
}

func (net *Network) lookup(target common.Hash, stopOnMatch bool) []*Node {
	var (
		asked          = make(map[NodeID]bool)
		seen           = make(map[NodeID]bool)
		reply          = make(chan []*Node, alpha)
		result         = nodesByDistance{target: target}
		pendingQueries = 0
	)
	// Get initial answers from the local node.
	result.push(net.tab.self, bucketSize)
	for {
		// Ask the α closest nodes that we haven't asked yet.
		for i := 0; i < len(result.entries) && pendingQueries < alpha; i++ {
			n := result.entries[i]
			if !asked[n.ID] {
				asked[n.ID] = true
				pendingQueries++
				net.reqQueryFindnode(n, target, reply)
			}
		}
		if pendingQueries == 0 {
			// We have asked all closest nodes, stop the search.
			break
		}
		// Wait for the next reply.
		for _, n := range <-reply {
			if n != nil && !seen[n.ID] {
				seen[n.ID] = true
				result.push(n, bucketSize)
				if stopOnMatch && n.sha == target {
					return result.entries
				}
			}
		}
		pendingQueries--
	}
	return result.entries
}

func (net *Network) RegisterTopic(topic Topic, stop <-chan struct{}) {
	select {
	case net.topicRegisterReq <- topicRegisterReq{true, topic}:
	case <-net.closed:
		return
	}
	select {
	case <-net.closed:
	case <-stop:
		select {
		case net.topicRegisterReq <- topicRegisterReq{false, topic}:
		case <-net.closed:
		}
	}
}

func (net *Network) SearchTopic(topic Topic, stop <-chan struct{}, found chan<- string) {
	select {
	case net.topicSearchReq <- topicSearchReq{topic, found}:
	case <-net.closed:
		return
	}
	select {
	case <-net.closed:
	case <-stop:
		select {
		case net.topicSearchReq <- topicSearchReq{topic, nil}:
		case <-net.closed:
		}
	}
}

func (net *Network) reqRefresh(nursery []*Node) <-chan struct{} {
	select {
	case net.refreshReq <- nursery:
		return <-net.refreshResp
	case <-net.closed:
		return net.closed
	}
}

func (net *Network) reqQueryFindnode(n *Node, target common.Hash, reply chan []*Node) bool {
	q := &findnodeQuery{remote: n, target: target, reply: reply}
	select {
	case net.queryReq <- q:
		return true
	case <-net.closed:
		return false
	}
}

func (net *Network) reqReadPacket(pkt ingressPacket) {
	select {
	case net.read <- pkt:
	case <-net.closed:
	}
}

func (net *Network) reqTableOp(f func()) (called bool) {
	select {
	case net.tableOpReq <- f:
		<-net.tableOpResp
		return true
	case <-net.closed:
		return false
	}
}

// TODO: external address handling.

func (net *Network) loop() {
	var (
		refreshTimer = time.NewTicker(autoRefreshInterval)
		refreshDone  chan struct{} // closed when the 'refresh' lookup has ended
	)

	// Tracking the next ticket to register.
	var (
		nextTicket        *ticketRef
		nextRegisterTimer *time.Timer
		nextRegisterTime  <-chan time.Time
	)
	defer func() {
		if nextRegisterTimer != nil {
			nextRegisterTimer.Stop()
		}
	}()
	resetNextTicket := func() {
		t, timeout := net.ticketStore.nextFilteredTicket()
		if t != nextTicket {
			nextTicket = t
			if nextRegisterTimer != nil {
				nextRegisterTimer.Stop()
				nextRegisterTime = nil
			}
			if t != nil {
				nextRegisterTimer = time.NewTimer(timeout)
				nextRegisterTime = nextRegisterTimer.C
			}
		}
	}

	// Tracking registration and search lookups.
	var (
		topicRegisterLookupTarget lookupInfo
		topicRegisterLookupDone   chan []*Node
		topicRegisterLookupTick   = time.NewTimer(0)
		topicSearchLookupTarget   lookupInfo
	)
	topicSearchLookupDone := make(chan []*Node, 1)
	<-topicRegisterLookupTick.C

	statsDump := time.NewTicker(10 * time.Second)

loop:
	for {
		resetNextTicket()

		select {
		case <-net.closeReq:
			debugLog("<-net.closeReq")
			break loop

		// Ingress packet handling.
		case pkt := <-net.read:
			debugLog("<-net.read")
			n := net.internNode(&pkt)
			prestate := n.state
			status := "ok"
			if err := net.handle(n, pkt.ev, &pkt); err != nil {
				status = err.Error()
			}
			if glog.V(logger.Detail) {
				glog.Infof("<<< (%d) %v from %x@%v: %v -> %v (%v)",
					net.tab.count, pkt.ev, pkt.remoteID[:8], pkt.remoteAddr, prestate, n.state, status)
			}
			// TODO: persist state if n.state goes >= known, delete if it goes <= known

		// State transition timeouts.
		case timeout := <-net.timeout:
			debugLog("<-net.timeout")
			if net.timeoutTimers[timeout] == nil {
				// Stale timer (was aborted).
				continue
			}
			delete(net.timeoutTimers, timeout)
			prestate := timeout.node.state
			status := "ok"
			if err := net.handle(timeout.node, timeout.ev, nil); err != nil {
				status = err.Error()
			}
			if glog.V(logger.Detail) {
				glog.Infof("--- (%d) %v for %x@%v: %v -> %v (%v)",
					net.tab.count, timeout.ev, timeout.node.ID[:8], timeout.node.addr(), prestate, timeout.node.state, status)
			}

		// Querying.
		case q := <-net.queryReq:
			debugLog("<-net.queryReq")
			if !q.start(net) {
				q.remote.deferQuery(q)
			}

		// Interacting with the table.
		case f := <-net.tableOpReq:
			debugLog("<-net.tableOpReq")
			f()
			net.tableOpResp <- struct{}{}

		// Topic registration stuff.
		case req := <-net.topicRegisterReq:
			debugLog("<-net.topicRegisterReq")
			if !req.add {
				net.ticketStore.removeRegisterTopic(req.topic)
				continue
			}
			net.ticketStore.addTopic(req.topic, true)
			// If we're currently waiting idle (nothing to look up), give the ticket store a
			// chance to start it sooner. This should speed up convergence of the radius
			// determination for new topics.
			// if topicRegisterLookupDone == nil {
			if topicRegisterLookupTarget.target == (common.Hash{}) {
				debugLog("topicRegisterLookupTarget == null")
				if topicRegisterLookupTick.Stop() {
					<-topicRegisterLookupTick.C
				}
				target, delay := net.ticketStore.nextRegisterLookup()
				topicRegisterLookupTarget = target
				topicRegisterLookupTick.Reset(delay)
			}

		case nodes := <-topicRegisterLookupDone:
			debugLog("<-topicRegisterLookupDone")
			net.ticketStore.registerLookupDone(topicRegisterLookupTarget, nodes, func(n *Node) []byte {
				net.ping(n, n.addr())
				return n.pingEcho
			})
			target, delay := net.ticketStore.nextRegisterLookup()
			topicRegisterLookupTarget = target
			topicRegisterLookupTick.Reset(delay)
			topicRegisterLookupDone = nil

		case <-topicRegisterLookupTick.C:
			debugLog("<-topicRegisterLookupTick")
			if (topicRegisterLookupTarget.target == common.Hash{}) {
				target, delay := net.ticketStore.nextRegisterLookup()
				topicRegisterLookupTarget = target
				topicRegisterLookupTick.Reset(delay)
				topicRegisterLookupDone = nil
			} else {
				topicRegisterLookupDone = make(chan []*Node)
				target := topicRegisterLookupTarget.target
				go func() { topicRegisterLookupDone <- net.lookup(target, false) }()
			}

		case <-nextRegisterTime:
			debugLog("<-nextRegisterTime")
			net.ticketStore.ticketRegistered(*nextTicket)
			//fmt.Println("sendTopicRegister", nextTicket.t.node.addr().String(), nextTicket.t.topics, nextTicket.idx, nextTicket.t.pong)
			net.conn.sendTopicRegister(nextTicket.t.node, nextTicket.t.topics, nextTicket.idx, nextTicket.t.pong)

		case req := <-net.topicSearchReq:
			debugLog("<-net.topicSearchReq")
			if req.found == nil {
				net.ticketStore.removeSearchTopic(req.topic)
				continue
			}
			net.ticketStore.addSearchTopic(req.topic, req.found)
			if (topicSearchLookupTarget.target == common.Hash{}) {
				topicSearchLookupDone <- nil
			}

		case nodes := <-topicSearchLookupDone:
			debugLog("<-topicSearchLookupDone")
			net.ticketStore.searchLookupDone(topicSearchLookupTarget, nodes, func(n *Node) []byte {
				net.ping(n, n.addr())
				return n.pingEcho
			}, func(n *Node, topic Topic) []byte {
				return net.conn.send(n, topicQueryPacket, topicQuery{Topic: topic}) // TODO: set expiration
			})
			topicSearchLookupTarget = net.ticketStore.nextSearchLookup()
			target := topicSearchLookupTarget.target
			if (target != common.Hash{}) {
				go func() { topicSearchLookupDone <- net.lookup(target, false) }()
			}

		case <-statsDump.C:
			debugLog("<-statsDump.C")
			/*r, ok := net.ticketStore.radius[testTopic]
			if !ok {
				fmt.Printf("(%x) no radius @ %v\n", net.tab.self.ID[:8], time.Now())
			} else {
				topics := len(net.ticketStore.tickets)
				tickets := len(net.ticketStore.nodes)
				rad := r.radius / (maxRadius/10000+1)
				fmt.Printf("(%x) topics:%d radius:%d tickets:%d @ %v\n", net.tab.self.ID[:8], topics, rad, tickets, time.Now())
			}*/

			tm := mclock.Now()
			for topic, r := range net.ticketStore.radius {
				if printTestImgLogs {
					rad := r.radius / (maxRadius/1000000 + 1)
					minrad := r.minRadius / (maxRadius/1000000 + 1)
					fmt.Printf("*R %d %v %016x %v\n", tm/1000000, topic, net.tab.self.sha[:8], rad)
					fmt.Printf("*MR %d %v %016x %v\n", tm/1000000, topic, net.tab.self.sha[:8], minrad)
				}
			}
			for topic, t := range net.topictab.topics {
				wp := t.wcl.nextWaitPeriod(tm)
				if printTestImgLogs {
					fmt.Printf("*W %d %v %016x %d\n", tm/1000000, topic, net.tab.self.sha[:8], wp/1000000)
				}
			}

		// Periodic / lookup-initiated bucket refresh.
		case <-refreshTimer.C:
			debugLog("<-refreshTimer.C")
			// TODO: ideally we would start the refresh timer after
			// fallback nodes have been set for the first time.
			if refreshDone == nil {
				refreshDone = make(chan struct{})
				net.refresh(refreshDone)
			}
		case doneChn := <-net.bucketFillChn:
			debugLog("bucketFill")
			net.bucketFill(doneChn)
		case newNursery := <-net.refreshReq:
			debugLog("<-net.refreshReq")
			if newNursery != nil {
				net.nursery = newNursery
			}
			if refreshDone == nil {
				refreshDone = make(chan struct{})
				net.refresh(refreshDone)
			}
			net.refreshResp <- refreshDone
		case <-refreshDone:
			debugLog("<-net.refreshDone")
			refreshDone = nil
		}
	}
	debugLog("loop stopped")

	glog.V(logger.Debug).Infof("shutting down")
	if net.conn != nil {
		net.conn.Close()
	}
	if refreshDone != nil {
		// TODO: wait for pending refresh.
		//<-refreshResults
	}
	// Cancel all pending timeouts.
	for _, timer := range net.timeoutTimers {
		timer.Stop()
	}
	if net.db != nil {
		net.db.close()
	}
	close(net.closed)
}

// Everything below runs on the Network.loop goroutine
// and can modify Node, Table and Network at any time without locking.

func (net *Network) refresh(done chan<- struct{}) {
	var seeds []*Node
	if net.db != nil {
		seeds = net.db.querySeeds(seedCount, seedMaxAge)
	}
	if len(seeds) == 0 {
		seeds = net.nursery
	}
	if len(seeds) == 0 {
		glog.V(logger.Detail).Info("no seed nodes found")
		close(done)
		return
	}
	for _, n := range seeds {
		if glog.V(logger.Debug) {
			var age string
			if net.db != nil {
				age = time.Since(net.db.lastPong(n.ID)).String()
			} else {
				age = "unknown"
			}
			glog.Infof("seed node (age %s): %v", age, n)
		}
		n = net.internNodeFromDB(n)
		if n.state == unknown {
			net.transition(n, verifyinit)
		}
		// Force-add the seed node so Lookup does something.
		// It will be deleted again if verification fails.
		net.tab.add(n)
	}
	// Start self lookup to fill up the buckets.
	go func() {
		net.Lookup(net.tab.self.ID)
		close(done)
	}()
}

func (net *Network) bucketFill(done chan<- struct{}) {
	target := net.tab.chooseBucketFillTarget()
	go func() {
		net.lookup(target, false)
		close(done)
	}()
}

func (net *Network) BucketFill() {
	done := make(chan struct{})
	select {
	case net.bucketFillChn <- done:
		<-done
	case <-net.closed:
		close(done)
	}
}

// Node Interning.

func (net *Network) internNode(pkt *ingressPacket) *Node {
	if n := net.nodes[pkt.remoteID]; n != nil {
		return n
	}
	n := NewNode(pkt.remoteID, pkt.remoteAddr.IP, uint16(pkt.remoteAddr.Port), uint16(pkt.remoteAddr.Port))
	n.state = unknown
	net.nodes[pkt.remoteID] = n
	return n
}

func (net *Network) internNodeFromDB(dbn *Node) *Node {
	if n := net.nodes[dbn.ID]; n != nil {
		return n
	}
	n := NewNode(dbn.ID, dbn.IP, dbn.UDP, dbn.TCP)
	n.state = unknown
	net.nodes[n.ID] = n
	return n
}

func (net *Network) internNodeFromNeighbours(rn rpcNode) (n *Node, err error) {
	if rn.ID == net.tab.self.ID {
		return nil, errors.New("is self")
	}
	n = net.nodes[rn.ID]
	if n == nil {
		// We haven't seen this node before.
		n, err = nodeFromRPC(rn)
		n.state = unknown
		if err == nil {
			net.nodes[n.ID] = n
		}
		return n, err
	}
	if !bytes.Equal(n.IP, rn.IP) || n.UDP != rn.UDP || n.TCP != rn.TCP {
		err = fmt.Errorf("metadata mismatch: got %v, want %v", rn, n)
	}
	return n, err
}

// nodeNetGuts is embedded in Node and contains fields.
type nodeNetGuts struct {
	// This is a cached copy of sha3(ID) which is used for node
	// distance calculations. This is part of Node in order to make it
	// possible to write tests that need a node at a certain distance.
	// In those tests, the content of sha will not actually correspond
	// with ID.
	sha common.Hash

	// State machine fields. Access to these fields
	// is restricted to the Network.loop goroutine.
	state             *nodeState
	pingEcho          []byte           // hash of last ping sent by us
	pingTopics        []Topic          // topic set sent by us in last ping
	deferredQueries   []*findnodeQuery // queries that can't be sent yet
	pendingNeighbours *findnodeQuery   // current query, waiting for reply
	queryTimeouts     int
}

func (n *nodeNetGuts) deferQuery(q *findnodeQuery) {
	n.deferredQueries = append(n.deferredQueries, q)
}

func (n *nodeNetGuts) startNextQuery(net *Network) {
	if len(n.deferredQueries) == 0 {
		return
	}
	nextq := n.deferredQueries[0]
	if nextq.start(net) {
		n.deferredQueries = append(n.deferredQueries[:0], n.deferredQueries[1:]...)
	}
}

func (q *findnodeQuery) start(net *Network) bool {
	// Satisfy queries against the local node directly.
	if q.remote == net.tab.self {
		closest := net.tab.closest(crypto.Keccak256Hash(q.target[:]), bucketSize)
		q.reply <- closest.entries
		return true
	}
	if q.remote.state.canQuery && q.remote.pendingNeighbours == nil {
		net.conn.sendFindnodeHash(q.remote, q.target)
		net.timedEvent(respTimeout, q.remote, neighboursTimeout)
		q.remote.pendingNeighbours = q
		return true
	}
	// If the node is not known yet, it won't accept queries.
	// Initiate the transition to known.
	// The request will be sent later when the node reaches known state.
	if q.remote.state == unknown {
		net.transition(q.remote, verifyinit)
	}
	return false
}

// Node Events (the input to the state machine).

type nodeEvent uint

//go:generate stringer -type=nodeEvent

const (
	invalidEvent nodeEvent = iota // zero is reserved

	// Packet type events.
	// These correspond to packet types in the UDP protocol.
	pingPacket
	pongPacket
	findnodePacket
	neighborsPacket
	findnodeHashPacket
	topicRegisterPacket
	topicQueryPacket
	topicNodesPacket

	// Non-packet events.
	// Event values in this category are allocated outside
	// the packet type range (packet types are encoded as a single byte).
	pongTimeout nodeEvent = iota + 256
	pingTimeout
	neighboursTimeout
)

// Node State Machine.

type nodeState struct {
	name     string
	handle   func(*Network, *Node, nodeEvent, *ingressPacket) (next *nodeState, err error)
	enter    func(*Network, *Node)
	canQuery bool
}

func (s *nodeState) String() string {
	return s.name
}

var (
	unknown          *nodeState
	verifyinit       *nodeState
	verifywait       *nodeState
	remoteverifywait *nodeState
	known            *nodeState
	contested        *nodeState
	unresponsive     *nodeState
)

func init() {
	unknown = &nodeState{
		name: "unknown",
		enter: func(net *Network, n *Node) {
			net.tab.delete(n)
			n.pingEcho = nil
			// Abort active queries.
			for _, q := range n.deferredQueries {
				q.reply <- nil
			}
			n.deferredQueries = nil
			if n.pendingNeighbours != nil {
				n.pendingNeighbours.reply <- nil
				n.pendingNeighbours = nil
			}
			n.queryTimeouts = 0
		},
		handle: func(net *Network, n *Node, ev nodeEvent, pkt *ingressPacket) (*nodeState, error) {
			switch ev {
			case pingPacket:
				net.handlePing(n, pkt)
				net.ping(n, pkt.remoteAddr)
				return verifywait, nil
			default:
				return unknown, errInvalidEvent
			}
		},
	}

	verifyinit = &nodeState{
		name: "verifyinit",
		enter: func(net *Network, n *Node) {
			net.ping(n, n.addr())
		},
		handle: func(net *Network, n *Node, ev nodeEvent, pkt *ingressPacket) (*nodeState, error) {
			switch ev {
			case pingPacket:
				net.handlePing(n, pkt)
				return verifywait, nil
			case pongPacket:
				err := net.handleKnownPong(n, pkt)
				return remoteverifywait, err
			case pongTimeout:
				return unknown, nil
			default:
				return verifyinit, errInvalidEvent
			}
		},
	}

	verifywait = &nodeState{
		name: "verifywait",
		handle: func(net *Network, n *Node, ev nodeEvent, pkt *ingressPacket) (*nodeState, error) {
			switch ev {
			case pongPacket:
				err := net.handleKnownPong(n, pkt)
				return known, err
			case pongTimeout:
				return unknown, nil
			default:
				return verifywait, errInvalidEvent
			}
		},
	}

	remoteverifywait = &nodeState{
		name: "remoteverifywait",
		enter: func(net *Network, n *Node) {
			net.timedEvent(respTimeout, n, pingTimeout)
		},
		handle: func(net *Network, n *Node, ev nodeEvent, pkt *ingressPacket) (*nodeState, error) {
			switch ev {
			case pingPacket:
				net.handlePing(n, pkt)
				return remoteverifywait, nil
			case pingTimeout:
				return known, nil
			default:
				return remoteverifywait, errInvalidEvent
			}
		},
	}

	known = &nodeState{
		name:     "known",
		canQuery: true,
		enter: func(net *Network, n *Node) {
			n.queryTimeouts = 0
			n.startNextQuery(net)
			// Insert into the table and start revalidation of the last node
			// in the bucket if it is full.
			last := net.tab.add(n)
			if last != nil && last.state == known {
				// TODO: do this asynchronously
				net.transition(last, contested)
			}
		},
		handle: func(net *Network, n *Node, ev nodeEvent, pkt *ingressPacket) (*nodeState, error) {
			switch ev {
			case pingPacket:
				net.handlePing(n, pkt)
				return known, nil
			case pongPacket:
				err := net.handleKnownPong(n, pkt)
				return known, err
			default:
				return net.handleQueryEvent(n, ev, pkt)
			}
		},
	}

	contested = &nodeState{
		name:     "contested",
		canQuery: true,
		enter: func(net *Network, n *Node) {
			net.ping(n, n.addr())
		},
		handle: func(net *Network, n *Node, ev nodeEvent, pkt *ingressPacket) (*nodeState, error) {
			switch ev {
			case pongPacket:
				// Node is still alive.
				err := net.handleKnownPong(n, pkt)
				return known, err
			case pongTimeout:
				net.tab.deleteReplace(n)
				return unresponsive, nil
			case pingPacket:
				net.handlePing(n, pkt)
				return contested, nil
			default:
				return net.handleQueryEvent(n, ev, pkt)
			}
		},
	}

	unresponsive = &nodeState{
		name:     "unresponsive",
		canQuery: true,
		handle: func(net *Network, n *Node, ev nodeEvent, pkt *ingressPacket) (*nodeState, error) {
			switch ev {
			case pingPacket:
				net.handlePing(n, pkt)
				return known, nil
			case pongPacket:
				err := net.handleKnownPong(n, pkt)
				return known, err
			default:
				return net.handleQueryEvent(n, ev, pkt)
			}
		},
	}
}

// handle processes packets sent by n and events related to n.
func (net *Network) handle(n *Node, ev nodeEvent, pkt *ingressPacket) error {
	if pkt != nil {
		if err := net.checkPacket(n, ev, pkt); err != nil {
			return err
		}
		// Start the background expiration goroutine after the first
		// successful communication. Subsequent calls have no effect if it
		// is already running. We do this here instead of somewhere else
		// so that the search for seed nodes also considers older nodes
		// that would otherwise be removed by the expirer.
		if net.db != nil {
			net.db.ensureExpirer()
		}
	}
	if n.state == nil {
		n.state = unknown //???
	}
	next, err := n.state.handle(net, n, ev, pkt)
	net.transition(n, next)
	return err
}

func (net *Network) checkPacket(n *Node, ev nodeEvent, pkt *ingressPacket) error {
	// Replay prevention checks.
	switch ev {
	case pingPacket, findnodeHashPacket, neighborsPacket:
		// TODO: check date is > last date seen
		// TODO: check ping version
	case pongPacket:
		if !bytes.Equal(pkt.data.(*pong).ReplyTok, n.pingEcho) {
			// fmt.Println("pong reply token mismatch")
			return fmt.Errorf("pong reply token mismatch")
		}
		n.pingEcho = nil
	}
	// Address validation.
	// TODO: Ideally we would do the following:
	//  - reject all packets with wrong address except ping.
	//  - for ping with new address, transition to verifywait but keep the
	//    previous node (with old address) around. if the new one reaches known,
	//    swap it out.
	return nil
}

func (net *Network) transition(n *Node, next *nodeState) {
	if n.state != next {
		n.state = next
		if next.enter != nil {
			next.enter(net, n)
		}
	}

	// TODO: persist/unpersist node
}

func (net *Network) timedEvent(d time.Duration, n *Node, ev nodeEvent) {
	timeout := timeoutEvent{ev, n}
	net.timeoutTimers[timeout] = time.AfterFunc(d, func() {
		select {
		case net.timeout <- timeout:
		case <-net.closed:
		}
	})
}

func (net *Network) abortTimedEvent(n *Node, ev nodeEvent) {
	timer := net.timeoutTimers[timeoutEvent{ev, n}]
	if timer != nil {
		timer.Stop()
		delete(net.timeoutTimers, timeoutEvent{ev, n})
	}
}

func (net *Network) ping(n *Node, addr *net.UDPAddr) {
	debugLog(fmt.Sprintf("ping(node = %x)", n.ID[:8]))
	n.pingTopics = net.ticketStore.regTopicSet()
	n.pingEcho = net.conn.sendPing(n, addr, n.pingTopics)
	net.timedEvent(respTimeout, n, pongTimeout)
}

func (net *Network) handlePing(n *Node, pkt *ingressPacket) {
	debugLog(fmt.Sprintf("handlePing(node = %x)", n.ID[:8]))
	ping := pkt.data.(*ping)
	n.TCP = ping.From.TCP
	t := net.topictab.getTicket(n, ping.Topics)

	pong := &pong{
		To:         makeEndpoint(n.addr(), n.TCP), // TODO: maybe use known TCP port from DB
		ReplyTok:   pkt.hash,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	}
	ticketToPong(t, pong)
	net.conn.send(n, pongPacket, pong)
}

func (net *Network) handleKnownPong(n *Node, pkt *ingressPacket) error {
	debugLog(fmt.Sprintf("handleKnownPong(node = %x)", n.ID[:8]))
	net.abortTimedEvent(n, pongTimeout)
	now := mclock.Now()
	ticket, err := pongToTicket(now, n.pingTopics, n, pkt)
	if err == nil {
		// fmt.Printf("(%x) ticket: %+v\n", net.tab.self.ID[:8], pkt.data)
		net.ticketStore.addTicket(now, pkt.data.(*pong).ReplyTok, ticket)
	} else {
		debugLog(fmt.Sprintf(" error: %v", err))
	}

	n.pingEcho = nil
	n.pingTopics = nil
	return err
}

func (net *Network) handleQueryEvent(n *Node, ev nodeEvent, pkt *ingressPacket) (*nodeState, error) {
	switch ev {
	case findnodePacket:
		target := crypto.Keccak256Hash(pkt.data.(*findnode).Target[:])
		results := net.tab.closest(target, bucketSize).entries
		net.conn.sendNeighbours(n, results)
		return n.state, nil
	case neighborsPacket:
		err := net.handleNeighboursPacket(n, pkt.data.(*neighbors))
		return n.state, err
	case neighboursTimeout:
		if n.pendingNeighbours != nil {
			n.pendingNeighbours.reply <- nil
			n.pendingNeighbours = nil
		}
		n.queryTimeouts++
		if n.queryTimeouts > maxFindnodeFailures && n.state == known {
			return contested, errors.New("too many timeouts")
		}
		return n.state, nil

	// v5

	case findnodeHashPacket:
		results := net.tab.closest(pkt.data.(*findnodeHash).Target, bucketSize).entries
		net.conn.sendNeighbours(n, results)
		return n.state, nil
	case topicRegisterPacket:
		//fmt.Println("got topicRegisterPacket")
		regdata := pkt.data.(*topicRegister)
		pong, err := net.checkTopicRegister(regdata)
		if err != nil {
			//fmt.Println(err)
			return n.state, fmt.Errorf("bad waiting ticket: %v", err)
		}
		net.topictab.useTicket(n, pong.TicketSerial, regdata.Topics, int(regdata.Idx), pong.Expiration, pong.WaitPeriods)
		return n.state, nil
	case topicQueryPacket:
		// TODO: handle expiration
		topic := pkt.data.(*topicQuery).Topic
		results := net.topictab.getEntries(topic)
		if _, ok := net.ticketStore.tickets[topic]; ok {
			results = append(results, net.tab.self) // we're not registering in our own table but if we're advertising, return ourselves too
		}
		if len(results) > 10 {
			results = results[:10]
		}
		var hash common.Hash
		copy(hash[:], pkt.hash)
		net.conn.sendTopicNodes(n, hash, results)
		return n.state, nil
	case topicNodesPacket:
		p := pkt.data.(*topicNodes)
		if net.ticketStore.gotTopicNodes(n, p.Echo, p.Nodes) {
			n.queryTimeouts++
			if n.queryTimeouts > maxFindnodeFailures && n.state == known {
				return contested, errors.New("too many timeouts")
			}
		}
		return n.state, nil

	default:
		return n.state, errInvalidEvent
	}
}

func (net *Network) checkTopicRegister(data *topicRegister) (*pong, error) {
	var pongpkt ingressPacket
	if err := decodePacket(data.Pong, &pongpkt); err != nil {
		return nil, err
	}
	if pongpkt.ev != pongPacket {
		return nil, errors.New("is not pong packet")
	}
	if pongpkt.remoteID != net.tab.self.ID {
		return nil, errors.New("not signed by us")
	}
	// check that we previously authorised all topics
	// that the other side is trying to register.
	if rlpHash(data.Topics) != pongpkt.data.(*pong).TopicHash {
		return nil, errors.New("topic hash mismatch")
	}
	if data.Idx < 0 || int(data.Idx) >= len(data.Topics) {
		return nil, errors.New("topic index out of range")
	}
	return pongpkt.data.(*pong), nil
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

func (net *Network) handleNeighboursPacket(n *Node, req *neighbors) error {
	if n.pendingNeighbours == nil {
		return errNoQuery
	}
	net.abortTimedEvent(n, neighboursTimeout)

	nodes := make([]*Node, len(req.Nodes))
	for i, rn := range req.Nodes {
		nn, err := net.internNodeFromNeighbours(rn)
		if err != nil {
			glog.V(logger.Debug).Infof("invalid neighbour from %x: %v", n.ID[:8], err)
			continue
		}
		nodes[i] = nn
		// Start validation of query results immediately.
		// This fills the table quickly.
		// TODO: generates way too many packets, maybe do it via queue.
		if nn.state == unknown {
			net.transition(nn, verifyinit)
		}
	}
	// TODO: don't ignore second packet
	n.pendingNeighbours.reply <- nodes
	n.pendingNeighbours = nil
	// Now that this query is done, start the next one.
	n.startNextQuery(net)
	return nil
}
