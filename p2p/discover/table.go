// Package discover implements the Node Discovery Protocol.
//
// The Node Discovery protocol provides a way to find RLPx nodes that
// can be connected to. It uses a Kademlia-like protocol to maintain a
// distributed database of the IDs and endpoints of all listening
// nodes.
package discover

import (
	"net"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

const (
	alpha               = 3              // Kademlia concurrency factor
	bucketSize          = 16             // Kademlia bucket size
	nBuckets            = nodeIDBits + 1 // Number of buckets
	maxBondingPingPongs = 10
)

type Table struct {
	mutex   sync.Mutex        // protects buckets, their content, and nursery
	buckets [nBuckets]*bucket // index of known nodes by distance
	nursery []*Node           // bootstrap nodes
	db      *nodeDB           // database of known nodes

	bondmu    sync.Mutex
	bonding   map[NodeID]*bondproc
	bondslots chan struct{} // limits total number of active bonding processes

	net  transport
	self *Node // metadata of the local node
}

type bondproc struct {
	err  error
	n    *Node
	done chan struct{}
}

// transport is implemented by the UDP transport.
// it is an interface so we can test without opening lots of UDP
// sockets and without generating a private key.
type transport interface {
	ping(NodeID, *net.UDPAddr) error
	waitping(NodeID) error
	findnode(toid NodeID, addr *net.UDPAddr, target NodeID) ([]*Node, error)
	close()
}

// bucket contains nodes, ordered by their last activity.
// the entry that was most recently active is the last element
// in entries.
type bucket struct {
	lastLookup time.Time
	entries    []*Node
}

func newTable(t transport, ourID NodeID, ourAddr *net.UDPAddr, nodeDBPath string) *Table {
	// If no seed cache was given, use an in-memory one
	db, err := newNodeDB(nodeDBPath)
	if err != nil {
		glog.V(logger.Warn).Infoln("Failed to open node database:", err)
		db, _ = newNodeDB("")
	}
	// Create the bootstrap table
	tab := &Table{
		net:       t,
		db:        db,
		self:      newNode(ourID, ourAddr),
		bonding:   make(map[NodeID]*bondproc),
		bondslots: make(chan struct{}, maxBondingPingPongs),
	}
	for i := 0; i < cap(tab.bondslots); i++ {
		tab.bondslots <- struct{}{}
	}
	for i := range tab.buckets {
		tab.buckets[i] = new(bucket)
	}
	return tab
}

// Self returns the local node.
func (tab *Table) Self() *Node {
	return tab.self
}

// Close terminates the network listener and flushes the seed cache.
func (tab *Table) Close() {
	tab.net.close()
	tab.db.close()
}

// Bootstrap sets the bootstrap nodes. These nodes are used to connect
// to the network if the table is empty. Bootstrap will also attempt to
// fill the table by performing random lookup operations on the
// network.
func (tab *Table) Bootstrap(nodes []*Node) {
	tab.mutex.Lock()
	// TODO: maybe filter nodes with bad fields (nil, etc.) to avoid strange crashes
	tab.nursery = make([]*Node, 0, len(nodes))
	for _, n := range nodes {
		cpy := *n
		tab.nursery = append(tab.nursery, &cpy)
	}
	tab.mutex.Unlock()
	tab.refresh()
}

// Lookup performs a network search for nodes close
// to the given target. It approaches the target by querying
// nodes that are closer to it on each iteration.
func (tab *Table) Lookup(target NodeID) []*Node {
	var (
		asked          = make(map[NodeID]bool)
		seen           = make(map[NodeID]bool)
		reply          = make(chan []*Node, alpha)
		pendingQueries = 0
	)
	// don't query further if we hit the target or ourself.
	// unlikely to happen often in practice.
	asked[target] = true
	asked[tab.self.ID] = true

	tab.mutex.Lock()
	// update last lookup stamp (for refresh logic)
	tab.buckets[logdist(tab.self.ID, target)].lastLookup = time.Now()
	// generate initial result set
	result := tab.closest(target, bucketSize)
	tab.mutex.Unlock()

	for {
		// ask the alpha closest nodes that we haven't asked yet
		for i := 0; i < len(result.entries) && pendingQueries < alpha; i++ {
			n := result.entries[i]
			if !asked[n.ID] {
				asked[n.ID] = true
				pendingQueries++
				go func() {
					r, _ := tab.net.findnode(n.ID, n.addr(), target)
					reply <- tab.bondall(r)
				}()
			}
		}
		if pendingQueries == 0 {
			// we have asked all closest nodes, stop the search
			break
		}
		// wait for the next reply
		for _, n := range <-reply {
			if n != nil && !seen[n.ID] {
				seen[n.ID] = true
				result.push(n, bucketSize)
			}
		}
		pendingQueries--
	}
	return result.entries
}

// refresh performs a lookup for a random target to keep buckets full.
func (tab *Table) refresh() {
	ld := -1 // logdist of chosen bucket
	tab.mutex.Lock()
	for i, b := range tab.buckets {
		if i > 0 && b.lastLookup.Before(time.Now().Add(-1*time.Hour)) {
			ld = i
			break
		}
	}
	tab.mutex.Unlock()

	result := tab.Lookup(randomID(tab.self.ID, ld))
	if len(result) == 0 {
		// Pick a batch of previously know seeds to lookup with
		seeds := tab.db.querySeeds(10)
		for _, seed := range seeds {
			glog.V(logger.Debug).Infoln("Seeding network with", seed)
		}
		// Bootstrap the table with a self lookup
		all := tab.bondall(append(tab.nursery, seeds...))
		tab.mutex.Lock()
		tab.add(all)
		tab.mutex.Unlock()
		tab.Lookup(tab.self.ID)
		// TODO: the Kademlia paper says that we're supposed to perform
		// random lookups in all buckets further away than our closest neighbor.
	}
}

// closest returns the n nodes in the table that are closest to the
// given id. The caller must hold tab.mutex.
func (tab *Table) closest(target NodeID, nresults int) *nodesByDistance {
	// This is a very wasteful way to find the closest nodes but
	// obviously correct. I believe that tree-based buckets would make
	// this easier to implement efficiently.
	close := &nodesByDistance{target: target}
	for _, b := range tab.buckets {
		for _, n := range b.entries {
			close.push(n, nresults)
		}
	}
	return close
}

func (tab *Table) len() (n int) {
	for _, b := range tab.buckets {
		n += len(b.entries)
	}
	return n
}

// bondall bonds with all given nodes concurrently and returns
// those nodes for which bonding has probably succeeded.
func (tab *Table) bondall(nodes []*Node) (result []*Node) {
	rc := make(chan *Node, len(nodes))
	for i := range nodes {
		go func(n *Node) {
			nn, _ := tab.bond(false, n.ID, n.addr(), uint16(n.TCPPort))
			rc <- nn
		}(nodes[i])
	}
	for _ = range nodes {
		if n := <-rc; n != nil {
			result = append(result, n)
		}
	}
	return result
}

// bond ensures the local node has a bond with the given remote node.
// It also attempts to insert the node into the table if bonding succeeds.
// The caller must not hold tab.mutex.
//
// A bond is must be established before sending findnode requests.
// Both sides must have completed a ping/pong exchange for a bond to
// exist. The total number of active bonding processes is limited in
// order to restrain network use.
//
// bond is meant to operate idempotently in that bonding with a remote
// node which still remembers a previously established bond will work.
// The remote node will simply not send a ping back, causing waitping
// to time out.
//
// If pinged is true, the remote node has just pinged us and one half
// of the process can be skipped.
func (tab *Table) bond(pinged bool, id NodeID, addr *net.UDPAddr, tcpPort uint16) (*Node, error) {
	var n *Node
	if n = tab.db.node(id); n == nil {
		tab.bondmu.Lock()
		w := tab.bonding[id]
		if w != nil {
			// Wait for an existing bonding process to complete.
			tab.bondmu.Unlock()
			<-w.done
		} else {
			// Register a new bonding process.
			w = &bondproc{done: make(chan struct{})}
			tab.bonding[id] = w
			tab.bondmu.Unlock()
			// Do the ping/pong. The result goes into w.
			tab.pingpong(w, pinged, id, addr, tcpPort)
			// Unregister the process after it's done.
			tab.bondmu.Lock()
			delete(tab.bonding, id)
			tab.bondmu.Unlock()
		}
		n = w.n
		if w.err != nil {
			return nil, w.err
		}
	}
	tab.mutex.Lock()
	defer tab.mutex.Unlock()
	if b := tab.buckets[logdist(tab.self.ID, n.ID)]; !b.bump(n) {
		tab.pingreplace(n, b)
	}
	return n, nil
}

func (tab *Table) pingpong(w *bondproc, pinged bool, id NodeID, addr *net.UDPAddr, tcpPort uint16) {
	// Request a bonding slot to limit network usage
	<-tab.bondslots
	defer func() { tab.bondslots <- struct{}{} }()

	// Ping the remote side and wait for a pong
	tab.db.updateLastPing(id, time.Now())
	if w.err = tab.net.ping(id, addr); w.err != nil {
		close(w.done)
		return
	}
	if !pinged {
		// Give the remote node a chance to ping us before we start
		// sending findnode requests. If they still remember us,
		// waitping will simply time out.
		tab.net.waitping(id)
	}
	// Bonding succeeded, update the node database
	w.n = &Node{
		ID:       id,
		IP:       addr.IP,
		DiscPort: addr.Port,
		TCPPort:  int(tcpPort),
	}
	tab.db.updateNode(w.n)
	tab.db.updateLastBond(id, time.Now())
	close(w.done)
}

func (tab *Table) pingreplace(new *Node, b *bucket) {
	if len(b.entries) == bucketSize {
		oldest := b.entries[bucketSize-1]
		if err := tab.net.ping(oldest.ID, oldest.addr()); err == nil {
			// The node responded, we don't need to replace it.
			return
		}
	} else {
		// Add a slot at the end so the last entry doesn't
		// fall off when adding the new node.
		b.entries = append(b.entries, nil)
	}
	copy(b.entries[1:], b.entries)
	b.entries[0] = new
}

// add puts the entries into the table if their corresponding
// bucket is not full. The caller must hold tab.mutex.
func (tab *Table) add(entries []*Node) {
outer:
	for _, n := range entries {
		if n == nil || n.ID == tab.self.ID {
			// skip bad entries. The RLP decoder returns nil for empty
			// input lists.
			continue
		}
		bucket := tab.buckets[logdist(tab.self.ID, n.ID)]
		for i := range bucket.entries {
			if bucket.entries[i].ID == n.ID {
				// already in bucket
				continue outer
			}
		}
		if len(bucket.entries) < bucketSize {
			bucket.entries = append(bucket.entries, n)
		}
	}
}

func (b *bucket) bump(n *Node) bool {
	for i := range b.entries {
		if b.entries[i].ID == n.ID {
			// move it to the front
			copy(b.entries[1:], b.entries[:i])
			b.entries[0] = n
			return true
		}
	}
	return false
}

// nodesByDistance is a list of nodes, ordered by
// distance to target.
type nodesByDistance struct {
	entries []*Node
	target  NodeID
}

// push adds the given node to the list, keeping the total size below maxElems.
func (h *nodesByDistance) push(n *Node, maxElems int) {
	ix := sort.Search(len(h.entries), func(i int) bool {
		return distcmp(h.target, h.entries[i].ID, n.ID) > 0
	})
	if len(h.entries) < maxElems {
		h.entries = append(h.entries, n)
	}
	if ix == len(h.entries) {
		// farther away than all nodes we already have.
		// if there was room for it, the node is now the last element.
	} else {
		// slide existing entries down to make room
		// this will overwrite the entry we just appended.
		copy(h.entries[ix+1:], h.entries[ix:])
		h.entries[ix] = n
	}
}
