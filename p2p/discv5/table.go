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

// Package discv5 implements the RLPx v5 Topic Discovery Protocol.
//
// The Topic Discovery protocol provides a way to find RLPx nodes that
// can be connected to. It uses a Kademlia-like protocol to maintain a
// distributed database of the IDs and endpoints of all listening
// nodes.
package discv5

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"sort"

	"github.com/ethereum/go-ethereum/common"
)

const (
	alpha      = 3  // Kademlia concurrency factor
	bucketSize = 16 // Kademlia bucket size
	hashBits   = len(common.Hash{}) * 8
	nBuckets   = hashBits + 1 // Number of buckets

	maxFindnodeFailures = 5
)

type Table struct {
	count         int               // number of nodes
	buckets       [nBuckets]*bucket // index of known nodes by distance
	nodeAddedHook func(*Node)       // for testing
	self          *Node             // metadata of the local node
}

// bucket contains nodes, ordered by their last activity. the entry
// that was most recently active is the first element in entries.
type bucket struct {
	entries      []*Node
	replacements []*Node
}

func newTable(ourID NodeID, ourAddr *net.UDPAddr) *Table {
	self := NewNode(ourID, ourAddr.IP, uint16(ourAddr.Port), uint16(ourAddr.Port))
	tab := &Table{self: self}
	for i := range tab.buckets {
		tab.buckets[i] = new(bucket)
	}
	return tab
}

const printTable = false

// chooseBucketRefreshTarget selects random refresh targets to keep all Kademlia
// buckets filled with live connections and keep the network topology healthy.
// This requires selecting addresses closer to our own with a higher probability
// in order to refresh closer buckets too.
//
// This algorithm approximates the distance distribution of existing nodes in the
// table by selecting a random node from the table and selecting a target address
// with a distance less than twice of that of the selected node.
// This algorithm will be improved later to specifically target the least recently
// used buckets.
func (tab *Table) chooseBucketRefreshTarget() common.Hash {
	entries := 0
	if printTable {
		fmt.Println()
	}
	for i, b := range tab.buckets {
		entries += len(b.entries)
		if printTable {
			for _, e := range b.entries {
				fmt.Println(i, e.state, e.addr().String(), e.ID.String(), e.sha.Hex())
			}
		}
	}

	prefix := binary.BigEndian.Uint64(tab.self.sha[0:8])
	dist := ^uint64(0)
	entry := int(randUint(uint32(entries + 1)))
	for _, b := range tab.buckets {
		if entry < len(b.entries) {
			n := b.entries[entry]
			dist = binary.BigEndian.Uint64(n.sha[0:8]) ^ prefix
			break
		}
		entry -= len(b.entries)
	}

	ddist := ^uint64(0)
	if dist+dist > dist {
		ddist = dist
	}
	targetPrefix := prefix ^ randUint64n(ddist)

	var target common.Hash
	binary.BigEndian.PutUint64(target[0:8], targetPrefix)
	rand.Read(target[8:])
	return target
}

// readRandomNodes fills the given slice with random nodes from the
// table. It will not write the same node more than once. The nodes in
// the slice are copies and can be modified by the caller.
func (tab *Table) readRandomNodes(buf []*Node) (n int) {
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
	if max < 2 {
		return 0
	}
	var b [4]byte
	rand.Read(b[:])
	return binary.BigEndian.Uint32(b[:]) % max
}

func randUint64n(max uint64) uint64 {
	if max < 2 {
		return 0
	}
	var b [8]byte
	rand.Read(b[:])
	return binary.BigEndian.Uint64(b[:]) % max
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

// add attempts to add the given node its corresponding bucket. If the
// bucket has space available, adding the node succeeds immediately.
// Otherwise, the node is added to the replacement cache for the bucket.
func (tab *Table) add(n *Node) (contested *Node) {
	//fmt.Println("add", n.addr().String(), n.ID.String(), n.sha.Hex())
	if n.ID == tab.self.ID {
		return
	}
	b := tab.buckets[logdist(tab.self.sha, n.sha)]
	switch {
	case b.bump(n):
		// n exists in b.
		return nil
	case len(b.entries) < bucketSize:
		// b has space available.
		b.addFront(n)
		tab.count++
		if tab.nodeAddedHook != nil {
			tab.nodeAddedHook(n)
		}
		return nil
	default:
		// b has no space left, add to replacement cache
		// and revalidate the last entry.
		// TODO: drop previous node
		b.replacements = append(b.replacements, n)
		if len(b.replacements) > bucketSize {
			copy(b.replacements, b.replacements[1:])
			b.replacements = b.replacements[:len(b.replacements)-1]
		}
		return b.entries[len(b.entries)-1]
	}
}

// stuff adds nodes the table to the end of their corresponding bucket
// if the bucket is not full.
func (tab *Table) stuff(nodes []*Node) {
outer:
	for _, n := range nodes {
		if n.ID == tab.self.ID {
			continue // don't add self
		}
		bucket := tab.buckets[logdist(tab.self.sha, n.sha)]
		for i := range bucket.entries {
			if bucket.entries[i].ID == n.ID {
				continue outer // already in bucket
			}
		}
		if len(bucket.entries) < bucketSize {
			bucket.entries = append(bucket.entries, n)
			tab.count++
			if tab.nodeAddedHook != nil {
				tab.nodeAddedHook(n)
			}
		}
	}
}

// delete removes an entry from the node table (used to evacuate
// failed/non-bonded discovery peers).
func (tab *Table) delete(node *Node) {
	//fmt.Println("delete", node.addr().String(), node.ID.String(), node.sha.Hex())
	bucket := tab.buckets[logdist(tab.self.sha, node.sha)]
	for i := range bucket.entries {
		if bucket.entries[i].ID == node.ID {
			bucket.entries = append(bucket.entries[:i], bucket.entries[i+1:]...)
			tab.count--
			return
		}
	}
}

func (tab *Table) deleteReplace(node *Node) {
	b := tab.buckets[logdist(tab.self.sha, node.sha)]
	i := 0
	for i < len(b.entries) {
		if b.entries[i].ID == node.ID {
			b.entries = append(b.entries[:i], b.entries[i+1:]...)
			tab.count--
		} else {
			i++
		}
	}
	// refill from replacement cache
	// TODO: maybe use random index
	if len(b.entries) < bucketSize && len(b.replacements) > 0 {
		ri := len(b.replacements) - 1
		b.addFront(b.replacements[ri])
		tab.count++
		b.replacements[ri] = nil
		b.replacements = b.replacements[:ri]
	}
}

func (b *bucket) addFront(n *Node) {
	b.entries = append(b.entries, nil)
	copy(b.entries[1:], b.entries)
	b.entries[0] = n
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
