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

package kademlia

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

const (
	bucketSize   = 4
	proxBinSize  = 2
	maxProx      = 8
	connRetryExp = 2
	maxPeers     = 100
)

var (
	purgeInterval        = 42 * time.Hour
	initialRetryInterval = 42 * time.Millisecond
	maxIdleInterval      = 42 * 1000 * time.Millisecond
	// maxIdleInterval      = 42 * 10	0 * time.Millisecond
)

type KadParams struct {
	// adjustable parameters
	MaxProx              int
	ProxBinSize          int
	BucketSize           int
	PurgeInterval        time.Duration
	InitialRetryInterval time.Duration
	MaxIdleInterval      time.Duration
	ConnRetryExp         int
}

func NewDefaultKadParams() *KadParams {
	return &KadParams{
		MaxProx:              maxProx,
		ProxBinSize:          proxBinSize,
		BucketSize:           bucketSize,
		PurgeInterval:        purgeInterval,
		InitialRetryInterval: initialRetryInterval,
		MaxIdleInterval:      maxIdleInterval,
		ConnRetryExp:         connRetryExp,
	}
}

// Kademlia is a table of active nodes
type Kademlia struct {
	addr       Address      // immutable baseaddress of the table
	*KadParams              // Kademlia configuration parameters
	proxLimit  int          // state, the PO of the first row of the most proximate bin
	proxSize   int          // state, the number of peers in the most proximate bin
	count      int          // number of active peers (w live connection)
	buckets    [][]Node     // the actual bins
	db         *KadDb       // kaddb, node record database
	lock       sync.RWMutex // mutex to access buckets
}

type Node interface {
	Addr() Address
	Url() string
	LastActive() time.Time
	Drop()
}

// public constructor
// add is the base address of the table
// params is KadParams configuration
func New(addr Address, params *KadParams) *Kademlia {
	buckets := make([][]Node, params.MaxProx+1)
	return &Kademlia{
		addr:      addr,
		KadParams: params,
		buckets:   buckets,
		db:        newKadDb(addr, params),
	}
}

// accessor for KAD base address
func (self *Kademlia) Addr() Address {
	return self.addr
}

// accessor for KAD active node count
func (self *Kademlia) Count() int {
	defer self.lock.Unlock()
	self.lock.Lock()
	return self.count
}

// accessor for KAD active node count
func (self *Kademlia) DBCount() int {
	return self.db.count()
}

// On is the entry point called when a new nodes is added
// unsafe in that node is not checked to be already active node (to be called once)
func (self *Kademlia) On(node Node, cb func(*NodeRecord, Node) error) (err error) {
	log.Debug(fmt.Sprintf("%v", self))
	defer self.lock.Unlock()
	self.lock.Lock()

	index := self.proximityBin(node.Addr())
	record := self.db.findOrCreate(index, node.Addr(), node.Url())

	if cb != nil {
		err = cb(record, node)
		log.Trace(fmt.Sprintf("cb(%v, %v) ->%v", record, node, err))
		if err != nil {
			return fmt.Errorf("unable to add node %v, callback error: %v", node.Addr(), err)
		}
		log.Debug(fmt.Sprintf("add node record %v with node %v", record, node))
	}

	// insert in kademlia table of active nodes
	bucket := self.buckets[index]
	// if bucket is full insertion replaces the worst node
	// TODO: give priority to peers with active traffic
	if len(bucket) < self.BucketSize { // >= allows us to add peers beyond the bucketsize limitation
		self.buckets[index] = append(bucket, node)
		log.Debug(fmt.Sprintf("add node %v to table", node))
		self.setProxLimit(index, true)
		record.node = node
		self.count++
		return nil
	}

	// always rotate peers
	idle := self.MaxIdleInterval
	var pos int
	var replaced Node
	for i, p := range bucket {
		idleInt := time.Since(p.LastActive())
		if idleInt > idle {
			idle = idleInt
			pos = i
			replaced = p
		}
	}
	if replaced == nil {
		log.Debug(fmt.Sprintf("all peers wanted, PO%03d bucket full", index))
		return fmt.Errorf("bucket full")
	}
	log.Debug(fmt.Sprintf("node %v replaced by %v (idle for %v  > %v)", replaced, node, idle, self.MaxIdleInterval))
	replaced.Drop()
	// actually replace in the row. When off(node) is called, the peer is no longer in the row
	bucket[pos] = node
	// there is no change in bucket cardinalities so no prox limit adjustment is needed
	record.node = node
	self.count++
	return nil

}

// Off is the called when a node is taken offline (from the protocol main loop exit)
func (self *Kademlia) Off(node Node, cb func(*NodeRecord, Node)) (err error) {
	self.lock.Lock()
	defer self.lock.Unlock()

	index := self.proximityBin(node.Addr())
	bucket := self.buckets[index]
	for i := 0; i < len(bucket); i++ {
		if node.Addr() == bucket[i].Addr() {
			self.buckets[index] = append(bucket[:i], bucket[(i+1):]...)
			self.setProxLimit(index, false)
			break
		}
	}

	record := self.db.index[node.Addr()]
	// callback on remove
	if cb != nil {
		cb(record, record.node)
	}
	record.node = nil
	self.count--
	log.Debug(fmt.Sprintf("remove node %v from table, population now is %v", node, self.count))

	return
}

// proxLimit is dynamically adjusted so that
// 1) there is no empty buckets in bin < proxLimit and
// 2) the sum of all items are the minimum possible but higher than ProxBinSize
// adjust Prox (proxLimit and proxSize after an insertion/removal of nodes)
// caller holds the lock
func (self *Kademlia) setProxLimit(r int, on bool) {
	// if the change is outside the core (PO lower)
	// and the change does not leave a bucket empty then
	// no adjustment needed
	if r < self.proxLimit && len(self.buckets[r]) > 0 {
		return
	}
	// if on=a node was added, then r must be within prox limit so increment cardinality
	if on {
		self.proxSize++
		curr := len(self.buckets[self.proxLimit])
		// if now core is big enough without the furthest bucket, then contract
		// this can result in more than one bucket change
		for self.proxSize >= self.ProxBinSize+curr && curr > 0 {
			self.proxSize -= curr
			self.proxLimit++
			curr = len(self.buckets[self.proxLimit])

			log.Trace(fmt.Sprintf("proxbin contraction (size: %v, limit: %v, bin: %v)", self.proxSize, self.proxLimit, r))
		}
		return
	}
	// otherwise
	if r >= self.proxLimit {
		self.proxSize--
	}
	// expand core by lowering prox limit until hit zero or cover the empty bucket or reached target cardinality
	for (self.proxSize < self.ProxBinSize || r < self.proxLimit) &&
		self.proxLimit > 0 {
		//
		self.proxLimit--
		self.proxSize += len(self.buckets[self.proxLimit])
		log.Trace(fmt.Sprintf("proxbin expansion (size: %v, limit: %v, bin: %v)", self.proxSize, self.proxLimit, r))
	}
}

/*
returns the list of nodes belonging to the same proximity bin
as the target. The most proximate bin will be the union of the bins between
proxLimit and MaxProx.
*/
func (self *Kademlia) FindClosest(target Address, max int) []Node {
	self.lock.Lock()
	defer self.lock.Unlock()

	r := nodesByDistance{
		target: target,
	}

	po := self.proximityBin(target)
	index := po
	step := 1
	log.Trace(fmt.Sprintf("serving %v nodes at %v (PO%02d)", max, index, po))

	// if max is set to 0, just want a full bucket, dynamic number
	min := max
	// set limit to max
	limit := max
	if max == 0 {
		min = 1
		limit = maxPeers
	}

	var n int
	for index >= 0 {
		// add entire bucket
		for _, p := range self.buckets[index] {
			r.push(p, limit)
			n++
		}
		// terminate if index reached the bottom or enough peers > min
		log.Trace(fmt.Sprintf("add %v -> %v (PO%02d, PO%03d)", len(self.buckets[index]), n, index, po))
		if n >= min && (step < 0 || max == 0) {
			break
		}
		// reach top most non-empty PO bucket, turn around
		if index == self.MaxProx {
			index = po
			step = -1
		}
		index += step
	}
	log.Trace(fmt.Sprintf("serve %d (<=%d) nodes for target lookup %v (PO%03d)", n, max, target, po))
	return r.nodes
}

func (self *Kademlia) Suggest() (*NodeRecord, bool, int) {
	defer self.lock.RUnlock()
	self.lock.RLock()
	return self.db.findBest(self.BucketSize, func(i int) int { return len(self.buckets[i]) })
}

//  adds node records to kaddb (persisted node record db)
func (self *Kademlia) Add(nrs []*NodeRecord) {
	self.db.add(nrs, self.proximityBin)
}

// nodesByDistance is a list of nodes, ordered by distance to target.
type nodesByDistance struct {
	nodes  []Node
	target Address
}

func sortedByDistanceTo(target Address, slice []Node) bool {
	var last Address
	for i, node := range slice {
		if i > 0 {
			if target.ProxCmp(node.Addr(), last) < 0 {
				return false
			}
		}
		last = node.Addr()
	}
	return true
}

// push(node, max) adds the given node to the list, keeping the total size
// below max elements.
func (h *nodesByDistance) push(node Node, max int) {
	// returns the firt index ix such that func(i) returns true
	ix := sort.Search(len(h.nodes), func(i int) bool {
		return h.target.ProxCmp(h.nodes[i].Addr(), node.Addr()) >= 0
	})

	if len(h.nodes) < max {
		h.nodes = append(h.nodes, node)
	}
	if ix < len(h.nodes) {
		copy(h.nodes[ix+1:], h.nodes[ix:])
		h.nodes[ix] = node
	}
}

/*
Taking the proximity order relative to a fix point x classifies the points in
the space (n byte long byte sequences) into bins. Items in each are at
most half as distant from x as items in the previous bin. Given a sample of
uniformly distributed items (a hash function over arbitrary sequence) the
proximity scale maps onto series of subsets with cardinalities on a negative
exponential scale.

It also has the property that any two item belonging to the same bin are at
most half as distant from each other as they are from x.

If we think of random sample of items in the bins as connections in a network of interconnected nodes than relative proximity can serve as the basis for local
decisions for graph traversal where the task is to find a route between two
points. Since in every hop, the finite distance halves, there is
a guaranteed constant maximum limit on the number of hops needed to reach one
node from the other.
*/

func (self *Kademlia) proximityBin(other Address) (ret int) {
	ret = proximity(self.addr, other)
	if ret > self.MaxProx {
		ret = self.MaxProx
	}
	return
}

// provides keyrange for chunk db iteration
func (self *Kademlia) KeyRange(other Address) (start, stop Address) {
	defer self.lock.RUnlock()
	self.lock.RLock()
	return KeyRange(self.addr, other, self.proxLimit)
}

// save persists kaddb on disk (written to file on path in json format.
func (self *Kademlia) Save(path string, cb func(*NodeRecord, Node)) error {
	return self.db.save(path, cb)
}

// Load(path) loads the node record database (kaddb) from file on path.
func (self *Kademlia) Load(path string, cb func(*NodeRecord, Node) error) (err error) {
	return self.db.load(path, cb)
}

// kademlia table + kaddb table displayed with ascii
func (self *Kademlia) String() string {
	defer self.lock.RUnlock()
	self.lock.RLock()
	defer self.db.lock.RUnlock()
	self.db.lock.RLock()

	var rows []string
	rows = append(rows, "=========================================================================")
	rows = append(rows, fmt.Sprintf("%v KΛÐΞMLIΛ hive: queen's address: %v", time.Now().UTC().Format(time.UnixDate), self.addr.String()[:6]))
	rows = append(rows, fmt.Sprintf("population: %d (%d), proxLimit: %d, proxSize: %d", self.count, len(self.db.index), self.proxLimit, self.proxSize))
	rows = append(rows, fmt.Sprintf("MaxProx: %d, ProxBinSize: %d, BucketSize: %d", self.MaxProx, self.ProxBinSize, self.BucketSize))

	for i, bucket := range self.buckets {

		if i == self.proxLimit {
			rows = append(rows, fmt.Sprintf("============ PROX LIMIT: %d ==========================================", i))
		}
		row := []string{fmt.Sprintf("%03d", i), fmt.Sprintf("%2d", len(bucket))}
		var k int
		c := self.db.cursors[i]
		for ; k < len(bucket); k++ {
			p := bucket[(c+k)%len(bucket)]
			row = append(row, p.Addr().String()[:6])
			if k == 4 {
				break
			}
		}
		for ; k < 4; k++ {
			row = append(row, "      ")
		}
		row = append(row, fmt.Sprintf("| %2d %2d", len(self.db.Nodes[i]), self.db.cursors[i]))

		for j, p := range self.db.Nodes[i] {
			row = append(row, p.Addr.String()[:6])
			if j == 3 {
				break
			}
		}
		rows = append(rows, strings.Join(row, " "))
		if i == self.MaxProx {
		}
	}
	rows = append(rows, "=========================================================================")
	return strings.Join(rows, "\n")
}
