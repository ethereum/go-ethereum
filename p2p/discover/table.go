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
)

const (
	alpha      = 3              // Kademlia concurrency factor
	bucketSize = 16             // Kademlia bucket size
	nBuckets   = nodeIDBits + 1 // Number of buckets
)

type Table struct {
	mutex   sync.Mutex        // protects buckets, their content, and nursery
	buckets [nBuckets]*bucket // index of known nodes by distance
	nursery []*Node           // bootstrap nodes

	net  transport
	self *Node // metadata of the local node
}

// transport is implemented by the UDP transport.
// it is an interface so we can test without opening lots of UDP
// sockets and without generating a private key.
type transport interface {
	ping(*Node) error
	findnode(e *Node, target NodeID) ([]*Node, error)
	close()
}

// bucket contains nodes, ordered by their last activity.
type bucket struct {
	lastLookup time.Time
	entries    []*Node
}

func newTable(t transport, ourID NodeID, ourAddr *net.UDPAddr) *Table {
	tab := &Table{net: t, self: newNode(ourID, ourAddr)}
	for i := range tab.buckets {
		tab.buckets[i] = new(bucket)
	}
	return tab
}

// Self returns the local node.
func (tab *Table) Self() *Node {
	return tab.self
}

// Close terminates the network listener.
func (tab *Table) Close() {
	tab.net.close()
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
					result, _ := tab.net.findnode(n, target)
					reply <- result
				}()
			}
		}
		if pendingQueries == 0 {
			// we have asked all closest nodes, stop the search
			break
		}

		// wait for the next reply
		for _, n := range <-reply {
			cn := n
			if !seen[n.ID] {
				seen[n.ID] = true
				result.push(cn, bucketSize)
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
		// bootstrap the table with a self lookup
		tab.mutex.Lock()
		tab.add(tab.nursery)
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

// bumpOrAdd updates the activity timestamp for the given node and
// attempts to insert the node into a bucket. The returned Node might
// not be part of the table. The caller must hold tab.mutex.
func (tab *Table) bumpOrAdd(node NodeID, from *net.UDPAddr) (n *Node) {
	b := tab.buckets[logdist(tab.self.ID, node)]
	if n = b.bump(node); n == nil {
		n = newNode(node, from)
		if len(b.entries) == bucketSize {
			tab.pingReplace(n, b)
		} else {
			b.entries = append(b.entries, n)
		}
	}
	return n
}

func (tab *Table) pingReplace(n *Node, b *bucket) {
	old := b.entries[bucketSize-1]
	go func() {
		if err := tab.net.ping(old); err == nil {
			// it responded, we don't need to replace it.
			return
		}
		// it didn't respond, replace the node if it is still the oldest node.
		tab.mutex.Lock()
		if len(b.entries) > 0 && b.entries[len(b.entries)-1] == old {
			// slide down other entries and put the new one in front.
			// TODO: insert in correct position to keep the order
			copy(b.entries[1:], b.entries)
			b.entries[0] = n
		}
		tab.mutex.Unlock()
	}()
}

// bump updates the activity timestamp for the given node.
// The caller must hold tab.mutex.
func (tab *Table) bump(node NodeID) {
	tab.buckets[logdist(tab.self.ID, node)].bump(node)
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

func (b *bucket) bump(id NodeID) *Node {
	for i, n := range b.entries {
		if n.ID == id {
			n.active = time.Now()
			// move it to the front
			copy(b.entries[1:], b.entries[:i+1])
			b.entries[0] = n
			return n
		}
	}
	return nil
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
