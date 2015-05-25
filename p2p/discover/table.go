// Package discover implements the Node Discovery Protocol.
//
// The Node Discovery protocol provides a way to find RLPx nodes that
// can be connected to. It uses a Kademlia-like protocol to maintain a
// distributed database of the IDs and endpoints of all listening
// nodes.
package discover

import (
	"crypto/rand"
	"encoding/binary"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

const (
	alpha      = 3  // Kademlia concurrency factor
	bucketSize = 16 // Kademlia bucket size
	hashBits   = len(common.Hash{}) * 8
	nBuckets   = hashBits + 1 // Number of buckets

	maxBondingPingPongs = 16
	maxFindnodeFailures = 5
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
	// If no node database was given, use an in-memory one
	db, err := newNodeDB(nodeDBPath, Version, ourID)
	if err != nil {
		glog.V(logger.Warn).Infoln("Failed to open node database:", err)
		db, _ = newNodeDB("", Version, ourID)
	}
	tab := &Table{
		net:       t,
		db:        db,
		self:      newNode(ourID, ourAddr.IP, uint16(ourAddr.Port), uint16(ourAddr.Port)),
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
// The returned node should not be modified by the caller.
func (tab *Table) Self() *Node {
	return tab.self
}

// ReadRandomNodes fills the given slice with random nodes from the
// table. It will not write the same node more than once. The nodes in
// the slice are copies and can be modified by the caller.
func (tab *Table) ReadRandomNodes(buf []*Node) (n int) {
	tab.mutex.Lock()
	defer tab.mutex.Unlock()
	// TODO: tree-based buckets would help here
	// Find all non-empty buckets and get a fresh slice of their entries.
	var buckets [][]*Node
	for _, b := range tab.buckets {
		if len(b.entries) > 0 {
			buckets = append(buckets, b.entries[:])
		}
	}
	if len(buckets) == 0 {
		return 0
	}
	// Shuffle the buckets.
	for i := uint32(len(buckets)) - 1; i > 0; i-- {
		j := randUint(i)
		buckets[i], buckets[j] = buckets[j], buckets[i]
	}
	// Move head of each bucket into buf, removing buckets that become empty.
	var i, j int
	for ; i < len(buf); i, j = i+1, (j+1)%len(buckets) {
		b := buckets[j]
		buf[i] = &(*b[0])
		buckets[j] = b[1:]
		if len(b) == 1 {
			buckets = append(buckets[:j], buckets[j+1:]...)
		}
		if len(buckets) == 0 {
			break
		}
	}
	return i + 1
}

func randUint(max uint32) uint32 {
	if max == 0 {
		return 0
	}
	var b [4]byte
	rand.Read(b[:])
	return binary.BigEndian.Uint32(b[:]) % max
}

// Close terminates the network listener and flushes the node database.
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
		cpy.sha = crypto.Sha3Hash(n.ID[:])
		tab.nursery = append(tab.nursery, &cpy)
	}
	tab.mutex.Unlock()
	tab.refresh()
}

// Lookup performs a network search for nodes close
// to the given target. It approaches the target by querying
// nodes that are closer to it on each iteration.
// The given target does not need to be an actual node
// identifier.
func (tab *Table) Lookup(targetID NodeID) []*Node {
	var (
		target         = crypto.Sha3Hash(targetID[:])
		asked          = make(map[NodeID]bool)
		seen           = make(map[NodeID]bool)
		reply          = make(chan []*Node, alpha)
		pendingQueries = 0
	)
	// don't query further if we hit ourself.
	// unlikely to happen often in practice.
	asked[tab.self.ID] = true

	tab.mutex.Lock()
	// update last lookup stamp (for refresh logic)
	tab.buckets[logdist(tab.self.sha, target)].lastLookup = time.Now()
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
					// Find potential neighbors to bond with
					r, err := tab.net.findnode(n.ID, n.addr(), targetID)
					if err != nil {
						// Bump the failure counter to detect and evacuate non-bonded entries
						fails := tab.db.findFails(n.ID) + 1
						tab.db.updateFindFails(n.ID, fails)
						glog.V(logger.Detail).Infof("Bumping failures for %x: %d", n.ID[:8], fails)

						if fails > maxFindnodeFailures {
							glog.V(logger.Detail).Infof("Evacuating node %x: %d findnode failures", n.ID[:8], fails)
							tab.del(n)
						}
					}
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
	// The Kademlia paper specifies that the bucket refresh should
	// perform a refresh in the least recently used bucket. We cannot
	// adhere to this because the findnode target is a 512bit value
	// (not hash-sized) and it is not easily possible to generate a
	// sha3 preimage that falls into a chosen bucket.
	//
	// We perform a lookup with a random target instead.
	var target NodeID
	rand.Read(target[:])
	result := tab.Lookup(target)
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
func (tab *Table) closest(target common.Hash, nresults int) *nodesByDistance {
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
			nn, _ := tab.bond(false, n.ID, n.addr(), uint16(n.TCP))
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
	// Retrieve a previously known node and any recent findnode failures
	node, fails := tab.db.node(id), 0
	if node != nil {
		fails = tab.db.findFails(id)
	}
	// If the node is unknown (non-bonded) or failed (remotely unknown), bond from scratch
	var result error
	if node == nil || fails > 0 {
		glog.V(logger.Detail).Infof("Bonding %x: known=%v, fails=%v", id[:8], node != nil, fails)

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
		// Retrieve the bonding results
		result = w.err
		if result == nil {
			node = w.n
		}
	}
	// Even if bonding temporarily failed, give the node a chance
	if node != nil {
		tab.mutex.Lock()
		defer tab.mutex.Unlock()

		b := tab.buckets[logdist(tab.self.sha, node.sha)]
		if !b.bump(node) {
			tab.pingreplace(node, b)
		}
		tab.db.updateFindFails(id, 0)
	}
	return node, result
}

func (tab *Table) pingpong(w *bondproc, pinged bool, id NodeID, addr *net.UDPAddr, tcpPort uint16) {
	// Request a bonding slot to limit network usage
	<-tab.bondslots
	defer func() { tab.bondslots <- struct{}{} }()

	// Ping the remote side and wait for a pong
	if w.err = tab.ping(id, addr); w.err != nil {
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
	w.n = newNode(id, addr.IP, uint16(addr.Port), tcpPort)
	tab.db.updateNode(w.n)
	close(w.done)
}

func (tab *Table) pingreplace(new *Node, b *bucket) {
	if len(b.entries) == bucketSize {
		oldest := b.entries[bucketSize-1]
		if err := tab.ping(oldest.ID, oldest.addr()); err == nil {
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

// ping a remote endpoint and wait for a reply, also updating the node database
// accordingly.
func (tab *Table) ping(id NodeID, addr *net.UDPAddr) error {
	// Update the last ping and send the message
	tab.db.updateLastPing(id, time.Now())
	if err := tab.net.ping(id, addr); err != nil {
		return err
	}
	// Pong received, update the database and return
	tab.db.updateLastPong(id, time.Now())
	tab.db.ensureExpirer()

	return nil
}

// add puts the entries into the table if their corresponding
// bucket is not full. The caller must hold tab.mutex.
func (tab *Table) add(entries []*Node) {
outer:
	for _, n := range entries {
		if n.ID == tab.self.ID {
			// don't add self.
			continue
		}
		bucket := tab.buckets[logdist(tab.self.sha, n.sha)]
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

// del removes an entry from the node table (used to evacuate failed/non-bonded
// discovery peers).
func (tab *Table) del(node *Node) {
	tab.mutex.Lock()
	defer tab.mutex.Unlock()

	bucket := tab.buckets[logdist(tab.self.sha, node.sha)]
	for i := range bucket.entries {
		if bucket.entries[i].ID == node.ID {
			bucket.entries = append(bucket.entries[:i], bucket.entries[i+1:]...)
			return
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
	target  common.Hash
}

// push adds the given node to the list, keeping the total size below maxElems.
func (h *nodesByDistance) push(n *Node, maxElems int) {
	ix := sort.Search(len(h.entries), func(i int) bool {
		return distcmp(h.target, h.entries[i].sha, n.sha) > 0
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
