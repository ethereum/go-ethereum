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

// Package discover implements the Node Discovery Protocol.
//
// The Node Discovery protocol provides a way to find RLPx nodes that
// can be connected to. It uses a Kademlia-like protocol to maintain a
// distributed database of the IDs and endpoints of all listening
// nodes.
package discover

import (
	"context"
	"fmt"
	"net/netip"
	"slices"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/netutil"
)

const (
	alpha           = 3  // Kademlia concurrency factor
	bucketSize      = 16 // Kademlia bucket size
	maxReplacements = 10 // Size of per-bucket replacement list

	// We keep buckets for the upper 1/15 of distances because
	// it's very unlikely we'll ever encounter a node that's closer.
	hashBits          = len(common.Hash{}) * 8
	nBuckets          = hashBits / 15       // Number of buckets
	bucketMinDistance = hashBits - nBuckets // Log distance of closest bucket

	// IP address limits.
	bucketIPLimit, bucketSubnet = 2, 24 // at most 2 addresses from the same /24
	tableIPLimit, tableSubnet   = 10, 24

	seedMinTableTime = 5 * time.Minute
	seedCount        = 30
	seedMaxAge       = 5 * 24 * time.Hour
)

// Table is the 'node table', a Kademlia-like index of neighbor nodes. The table keeps
// itself up-to-date by verifying the liveness of neighbors and requesting their node
// records when announcements of a new record version are received.
type Table struct {
	mutex        sync.Mutex        // protects buckets, bucket content, nursery, rand
	buckets      [nBuckets]*bucket // index of known nodes by distance
	nursery      []*enode.Node     // bootstrap nodes
	rand         reseedingRandom   // source of randomness, periodically reseeded
	ips          netutil.DistinctNetSet
	revalidation tableRevalidation

	db  *enode.DB // database of known nodes
	net transport
	cfg Config
	log log.Logger

	// loop channels
	refreshReq      chan chan struct{}
	revalResponseCh chan revalidationResponse
	addNodeCh       chan addNodeOp
	addNodeHandled  chan bool
	trackRequestCh  chan trackRequestOp
	initDone        chan struct{}
	closeReq        chan struct{}
	closed          chan struct{}

	nodeAddedHook   func(*bucket, *tableNode)
	nodeRemovedHook func(*bucket, *tableNode)
}

// transport is implemented by the UDP transports.
type transport interface {
	Self() *enode.Node
	RequestENR(*enode.Node) (*enode.Node, error)
	lookupRandom() []*enode.Node
	lookupSelf() []*enode.Node
	ping(*enode.Node) (seq uint64, err error)
}

// bucket contains nodes, ordered by their last activity. the entry
// that was most recently active is the first element in entries.
type bucket struct {
	entries      []*tableNode // live entries, sorted by time of last contact
	replacements []*tableNode // recently seen nodes to be used if revalidation fails
	ips          netutil.DistinctNetSet
	index        int
}

type addNodeOp struct {
	node         *enode.Node
	isInbound    bool
	forceSetLive bool // for tests
}

type trackRequestOp struct {
	node       *enode.Node
	foundNodes []*enode.Node
	success    bool
}

func newTable(t transport, db *enode.DB, cfg Config) (*Table, error) {
	cfg = cfg.withDefaults()
	tab := &Table{
		net:             t,
		db:              db,
		cfg:             cfg,
		log:             cfg.Log,
		refreshReq:      make(chan chan struct{}),
		revalResponseCh: make(chan revalidationResponse),
		addNodeCh:       make(chan addNodeOp),
		addNodeHandled:  make(chan bool),
		trackRequestCh:  make(chan trackRequestOp),
		initDone:        make(chan struct{}),
		closeReq:        make(chan struct{}),
		closed:          make(chan struct{}),
		ips:             netutil.DistinctNetSet{Subnet: tableSubnet, Limit: tableIPLimit},
	}
	for i := range tab.buckets {
		tab.buckets[i] = &bucket{
			index: i,
			ips:   netutil.DistinctNetSet{Subnet: bucketSubnet, Limit: bucketIPLimit},
		}
	}
	tab.rand.seed()
	tab.revalidation.init(&cfg)

	// initial table content
	if err := tab.setFallbackNodes(cfg.Bootnodes); err != nil {
		return nil, err
	}
	tab.loadSeedNodes()

	return tab, nil
}

// Nodes returns all nodes contained in the table.
func (tab *Table) Nodes() [][]BucketNode {
	tab.mutex.Lock()
	defer tab.mutex.Unlock()

	nodes := make([][]BucketNode, len(tab.buckets))
	for i, b := range &tab.buckets {
		nodes[i] = make([]BucketNode, len(b.entries))
		for j, n := range b.entries {
			nodes[i][j] = BucketNode{
				Node:          n.Node,
				Checks:        int(n.livenessChecks),
				Live:          n.isValidatedLive,
				AddedToTable:  n.addedToTable,
				AddedToBucket: n.addedToBucket,
			}
		}
	}
	return nodes
}

func (tab *Table) self() *enode.Node {
	return tab.net.Self()
}

// getNode returns the node with the given ID or nil if it isn't in the table.
func (tab *Table) getNode(id enode.ID) *enode.Node {
	tab.mutex.Lock()
	defer tab.mutex.Unlock()

	b := tab.bucket(id)
	for _, e := range b.entries {
		if e.ID() == id {
			return e.Node
		}
	}
	return nil
}

// close terminates the network listener and flushes the node database.
func (tab *Table) close() {
	close(tab.closeReq)
	<-tab.closed
}

// setFallbackNodes sets the initial points of contact. These nodes
// are used to connect to the network if the table is empty and there
// are no known nodes in the database.
func (tab *Table) setFallbackNodes(nodes []*enode.Node) error {
	nursery := make([]*enode.Node, 0, len(nodes))
	for _, n := range nodes {
		if err := n.ValidateComplete(); err != nil {
			return fmt.Errorf("bad bootstrap node %q: %v", n, err)
		}
		if tab.cfg.NetRestrict != nil && !tab.cfg.NetRestrict.ContainsAddr(n.IPAddr()) {
			tab.log.Error("Bootstrap node filtered by netrestrict", "id", n.ID(), "ip", n.IPAddr())
			continue
		}
		nursery = append(nursery, n)
	}
	tab.nursery = nursery
	return nil
}

// isInitDone returns whether the table's initial seeding procedure has completed.
func (tab *Table) isInitDone() bool {
	select {
	case <-tab.initDone:
		return true
	default:
		return false
	}
}

func (tab *Table) refresh() <-chan struct{} {
	done := make(chan struct{})
	select {
	case tab.refreshReq <- done:
	case <-tab.closeReq:
		close(done)
	}
	return done
}

// findnodeByID returns the n nodes in the table that are closest to the given id.
// This is used by the FINDNODE/v4 handler.
//
// The preferLive parameter says whether the caller wants liveness-checked results. If
// preferLive is true and the table contains any verified nodes, the result will not
// contain unverified nodes. However, if there are no verified nodes at all, the result
// will contain unverified nodes.
func (tab *Table) findnodeByID(target enode.ID, nresults int, preferLive bool) *nodesByDistance {
	tab.mutex.Lock()
	defer tab.mutex.Unlock()

	// Scan all buckets. There might be a better way to do this, but there aren't that many
	// buckets, so this solution should be fine. The worst-case complexity of this loop
	// is O(tab.len() * nresults).
	nodes := &nodesByDistance{target: target}
	liveNodes := &nodesByDistance{target: target}
	for _, b := range &tab.buckets {
		for _, n := range b.entries {
			nodes.push(n.Node, nresults)
			if preferLive && n.isValidatedLive {
				liveNodes.push(n.Node, nresults)
			}
		}
	}

	if preferLive && len(liveNodes.entries) > 0 {
		return liveNodes
	}
	return nodes
}

// appendLiveNodes adds nodes at the given distance to the result slice.
// This is used by the FINDNODE/v5 handler.
func (tab *Table) appendLiveNodes(dist uint, result []*enode.Node) []*enode.Node {
	if dist > 256 {
		return result
	}
	if dist == 0 {
		return append(result, tab.self())
	}

	tab.mutex.Lock()
	for _, n := range tab.bucketAtDistance(int(dist)).entries {
		if n.isValidatedLive {
			result = append(result, n.Node)
		}
	}
	tab.mutex.Unlock()

	// Shuffle result to avoid always returning same nodes in FINDNODE/v5.
	tab.rand.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})
	return result
}

// len returns the number of nodes in the table.
func (tab *Table) len() (n int) {
	tab.mutex.Lock()
	defer tab.mutex.Unlock()

	for _, b := range &tab.buckets {
		n += len(b.entries)
	}
	return n
}

// addFoundNode adds a node which may not be live. If the bucket has space available,
// adding the node succeeds immediately. Otherwise, the node is added to the replacements
// list.
//
// The caller must not hold tab.mutex.
func (tab *Table) addFoundNode(n *enode.Node, forceSetLive bool) bool {
	op := addNodeOp{node: n, isInbound: false, forceSetLive: forceSetLive}
	select {
	case tab.addNodeCh <- op:
		return <-tab.addNodeHandled
	case <-tab.closeReq:
		return false
	}
}

// addInboundNode adds a node from an inbound contact. If the bucket has no space, the
// node is added to the replacements list.
//
// There is an additional safety measure: if the table is still initializing the node is
// not added. This prevents an attack where the table could be filled by just sending ping
// repeatedly.
//
// The caller must not hold tab.mutex.
func (tab *Table) addInboundNode(n *enode.Node) bool {
	op := addNodeOp{node: n, isInbound: true}
	select {
	case tab.addNodeCh <- op:
		return <-tab.addNodeHandled
	case <-tab.closeReq:
		return false
	}
}

func (tab *Table) trackRequest(n *enode.Node, success bool, foundNodes []*enode.Node) {
	op := trackRequestOp{n, foundNodes, success}
	select {
	case tab.trackRequestCh <- op:
	case <-tab.closeReq:
	}
}

// loop is the main loop of Table.
func (tab *Table) loop() {
	var (
		refresh         = time.NewTimer(tab.nextRefreshTime())
		refreshDone     = make(chan struct{})           // where doRefresh reports completion
		waiting         = []chan struct{}{tab.initDone} // holds waiting callers while doRefresh runs
		revalTimer      = mclock.NewAlarm(tab.cfg.Clock)
		reseedRandTimer = time.NewTicker(10 * time.Minute)
	)
	defer refresh.Stop()
	defer revalTimer.Stop()
	defer reseedRandTimer.Stop()

	// Start initial refresh.
	go tab.doRefresh(refreshDone)

loop:
	for {
		nextTime := tab.revalidation.run(tab, tab.cfg.Clock.Now())
		revalTimer.Schedule(nextTime)

		select {
		case <-reseedRandTimer.C:
			tab.rand.seed()

		case <-revalTimer.C():

		case r := <-tab.revalResponseCh:
			tab.revalidation.handleResponse(tab, r)

		case op := <-tab.addNodeCh:
			tab.mutex.Lock()
			ok := tab.handleAddNode(op)
			tab.mutex.Unlock()
			tab.addNodeHandled <- ok

		case op := <-tab.trackRequestCh:
			tab.handleTrackRequest(op)

		case <-refresh.C:
			if refreshDone == nil {
				refreshDone = make(chan struct{})
				go tab.doRefresh(refreshDone)
			}

		case req := <-tab.refreshReq:
			waiting = append(waiting, req)
			if refreshDone == nil {
				refreshDone = make(chan struct{})
				go tab.doRefresh(refreshDone)
			}

		case <-refreshDone:
			for _, ch := range waiting {
				close(ch)
			}
			waiting, refreshDone = nil, nil
			refresh.Reset(tab.nextRefreshTime())

		case <-tab.closeReq:
			break loop
		}
	}

	if refreshDone != nil {
		<-refreshDone
	}
	for _, ch := range waiting {
		close(ch)
	}
	close(tab.closed)
}

// doRefresh performs a lookup for a random target to keep buckets full. seed nodes are
// inserted if the table is empty (initial bootstrap or discarded faulty peers).
func (tab *Table) doRefresh(done chan struct{}) {
	defer close(done)

	// Load nodes from the database and insert
	// them. This should yield a few previously seen nodes that are
	// (hopefully) still alive.
	tab.loadSeedNodes()

	// Run self lookup to discover new neighbor nodes.
	tab.net.lookupSelf()

	// The Kademlia paper specifies that the bucket refresh should
	// perform a lookup in the least recently used bucket. We cannot
	// adhere to this because the findnode target is a 512bit value
	// (not hash-sized) and it is not easily possible to generate a
	// sha3 preimage that falls into a chosen bucket.
	// We perform a few lookups with a random target instead.
	for i := 0; i < 3; i++ {
		tab.net.lookupRandom()
	}
}

func (tab *Table) loadSeedNodes() {
	seeds := tab.db.QuerySeeds(seedCount, seedMaxAge)
	seeds = append(seeds, tab.nursery...)
	for i := range seeds {
		seed := seeds[i]
		if tab.log.Enabled(context.Background(), log.LevelTrace) {
			age := time.Since(tab.db.LastPongReceived(seed.ID(), seed.IPAddr()))
			addr, _ := seed.UDPEndpoint()
			tab.log.Trace("Found seed node in database", "id", seed.ID(), "addr", addr, "age", age)
		}
		tab.mutex.Lock()
		tab.handleAddNode(addNodeOp{node: seed, isInbound: false})
		tab.mutex.Unlock()
	}
}

func (tab *Table) nextRefreshTime() time.Duration {
	half := tab.cfg.RefreshInterval / 2
	return half + time.Duration(tab.rand.Int63n(int64(half)))
}

// bucket returns the bucket for the given node ID hash.
func (tab *Table) bucket(id enode.ID) *bucket {
	d := enode.LogDist(tab.self().ID(), id)
	return tab.bucketAtDistance(d)
}

func (tab *Table) bucketAtDistance(d int) *bucket {
	if d <= bucketMinDistance {
		return tab.buckets[0]
	}
	return tab.buckets[d-bucketMinDistance-1]
}

func (tab *Table) addIP(b *bucket, ip netip.Addr) bool {
	if !ip.IsValid() || ip.IsUnspecified() {
		return false // Nodes without IP cannot be added.
	}
	if netutil.AddrIsLAN(ip) {
		return true
	}
	if !tab.ips.AddAddr(ip) {
		tab.log.Debug("IP exceeds table limit", "ip", ip)
		return false
	}
	if !b.ips.AddAddr(ip) {
		tab.log.Debug("IP exceeds bucket limit", "ip", ip)
		tab.ips.RemoveAddr(ip)
		return false
	}
	return true
}

func (tab *Table) removeIP(b *bucket, ip netip.Addr) {
	if netutil.AddrIsLAN(ip) {
		return
	}
	tab.ips.RemoveAddr(ip)
	b.ips.RemoveAddr(ip)
}

// handleAddNode adds the node in the request to the table, if there is space.
// The caller must hold tab.mutex.
func (tab *Table) handleAddNode(req addNodeOp) bool {
	if req.node.ID() == tab.self().ID() {
		return false
	}
	// For nodes from inbound contact, there is an additional safety measure: if the table
	// is still initializing the node is not added.
	if req.isInbound && !tab.isInitDone() {
		return false
	}

	b := tab.bucket(req.node.ID())
	n, _ := tab.bumpInBucket(b, req.node, req.isInbound)
	if n != nil {
		// Already in bucket.
		return false
	}
	if len(b.entries) >= bucketSize {
		// Bucket full, maybe add as replacement.
		tab.addReplacement(b, req.node)
		return false
	}
	if !tab.addIP(b, req.node.IPAddr()) {
		// Can't add: IP limit reached.
		return false
	}

	// Add to bucket.
	wn := &tableNode{Node: req.node}
	if req.forceSetLive {
		wn.livenessChecks = 1
		wn.isValidatedLive = true
	}
	b.entries = append(b.entries, wn)
	b.replacements = deleteNode(b.replacements, wn.ID())
	tab.nodeAdded(b, wn)
	return true
}

// addReplacement adds n to the replacement cache of bucket b.
func (tab *Table) addReplacement(b *bucket, n *enode.Node) {
	if containsID(b.replacements, n.ID()) {
		// TODO: update ENR
		return
	}
	if !tab.addIP(b, n.IPAddr()) {
		return
	}

	wn := &tableNode{Node: n, addedToTable: time.Now()}
	var removed *tableNode
	b.replacements, removed = pushNode(b.replacements, wn, maxReplacements)
	if removed != nil {
		tab.removeIP(b, removed.IPAddr())
	}
}

func (tab *Table) nodeAdded(b *bucket, n *tableNode) {
	if n.addedToTable == (time.Time{}) {
		n.addedToTable = time.Now()
	}
	n.addedToBucket = time.Now()
	tab.revalidation.nodeAdded(tab, n)
	if tab.nodeAddedHook != nil {
		tab.nodeAddedHook(b, n)
	}
	if metrics.Enabled {
		bucketsCounter[b.index].Inc(1)
	}
}

func (tab *Table) nodeRemoved(b *bucket, n *tableNode) {
	tab.revalidation.nodeRemoved(n)
	if tab.nodeRemovedHook != nil {
		tab.nodeRemovedHook(b, n)
	}
	if metrics.Enabled {
		bucketsCounter[b.index].Dec(1)
	}
}

// deleteInBucket removes node n from the table.
// If there are replacement nodes in the bucket, the node is replaced.
func (tab *Table) deleteInBucket(b *bucket, id enode.ID) *tableNode {
	index := slices.IndexFunc(b.entries, func(e *tableNode) bool { return e.ID() == id })
	if index == -1 {
		// Entry has been removed already.
		return nil
	}

	// Remove the node.
	n := b.entries[index]
	b.entries = slices.Delete(b.entries, index, index+1)
	tab.removeIP(b, n.IPAddr())
	tab.nodeRemoved(b, n)

	// Add replacement.
	if len(b.replacements) == 0 {
		tab.log.Debug("Removed dead node", "b", b.index, "id", n.ID(), "ip", n.IPAddr())
		return nil
	}
	rindex := tab.rand.Intn(len(b.replacements))
	rep := b.replacements[rindex]
	b.replacements = slices.Delete(b.replacements, rindex, rindex+1)
	b.entries = append(b.entries, rep)
	tab.nodeAdded(b, rep)
	tab.log.Debug("Replaced dead node", "b", b.index, "id", n.ID(), "ip", n.IPAddr(), "r", rep.ID(), "rip", rep.IPAddr())
	return rep
}

// bumpInBucket updates a node record if it exists in the bucket.
// The second return value reports whether the node's endpoint (IP/port) was updated.
func (tab *Table) bumpInBucket(b *bucket, newRecord *enode.Node, isInbound bool) (n *tableNode, endpointChanged bool) {
	i := slices.IndexFunc(b.entries, func(elem *tableNode) bool {
		return elem.ID() == newRecord.ID()
	})
	if i == -1 {
		return nil, false // not in bucket
	}
	n = b.entries[i]

	// For inbound updates (from the node itself) we accept any change, even if it sets
	// back the sequence number. For found nodes (!isInbound), seq has to advance. Note
	// this check also ensures found discv4 nodes (which always have seq=0) can't be
	// updated.
	if newRecord.Seq() <= n.Seq() && !isInbound {
		return n, false
	}

	// Check endpoint update against IP limits.
	ipchanged := newRecord.IPAddr() != n.IPAddr()
	portchanged := newRecord.UDP() != n.UDP()
	if ipchanged {
		tab.removeIP(b, n.IPAddr())
		if !tab.addIP(b, newRecord.IPAddr()) {
			// It doesn't fit with the limit, put the previous record back.
			tab.addIP(b, n.IPAddr())
			return n, false
		}
	}

	// Apply update.
	n.Node = newRecord
	if ipchanged || portchanged {
		// Ensure node is revalidated quickly for endpoint changes.
		tab.revalidation.nodeEndpointChanged(tab, n)
		return n, true
	}
	return n, false
}

func (tab *Table) handleTrackRequest(op trackRequestOp) {
	var fails int
	if op.success {
		// Reset failure counter because it counts _consecutive_ failures.
		tab.db.UpdateFindFails(op.node.ID(), op.node.IPAddr(), 0)
	} else {
		fails = tab.db.FindFails(op.node.ID(), op.node.IPAddr())
		fails++
		tab.db.UpdateFindFails(op.node.ID(), op.node.IPAddr(), fails)
	}

	tab.mutex.Lock()
	defer tab.mutex.Unlock()

	b := tab.bucket(op.node.ID())
	// Remove the node from the local table if it fails to return anything useful too
	// many times, but only if there are enough other nodes in the bucket. This latter
	// condition specifically exists to make bootstrapping in smaller test networks more
	// reliable.
	if fails >= maxFindnodeFailures && len(b.entries) >= bucketSize/4 {
		tab.deleteInBucket(b, op.node.ID())
	}

	// Add found nodes.
	for _, n := range op.foundNodes {
		tab.handleAddNode(addNodeOp{n, false, false})
	}
}

// pushNode adds n to the front of list, keeping at most max items.
func pushNode(list []*tableNode, n *tableNode, max int) ([]*tableNode, *tableNode) {
	if len(list) < max {
		list = append(list, nil)
	}
	removed := list[len(list)-1]
	copy(list[1:], list)
	list[0] = n
	return list, removed
}
