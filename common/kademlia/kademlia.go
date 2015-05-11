package kademlia

import (
	"fmt"
	"sort"
	// "math"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
)

var kadlogger = logger.NewLogger("KΛÐ")

const (
	bucketSize = 20
	maxProx    = 255
)

type Kademlia struct {
	// immutable baseparam
	addr Address

	// adjustable parameters
	BucketSize     int
	MaxProx        int
	MaxProxBinSize int
	nodeDB         [][]*nodeRecord
	nodeIndex      map[Address]*nodeRecord

	GetNode func(int)

	// state
	proxLimit int
	proxSize  int

	//
	count   int
	buckets []*bucket

	lock  sync.RWMutex
	quitC chan bool
}

type Address common.Hash

type Node interface {
	Addr() Address
	// Url()
	LastActive() time.Time
	Drop()
}

type nodeRecord struct {
	Address Address `json:address`
	Active  int64   `json:active`
	node    Node
}

func (self *nodeRecord) setActive() {
	if self.node != nil {
		self.Active = self.node.LastActive().UnixNano()
	}
}

type kadDB struct {
	Address Address         `json:address`
	Nodes   [][]*nodeRecord `json:nodes`
}

// public constructor with compulsory arguments
// hash is a byte slice of length equal to self.HashBytes
func New(a Address) *Kademlia {
	return &Kademlia{
		addr: a, // compulsory fields without default
	}
}

// accessor for KAD self address
func (self *Kademlia) Addr() Address {
	return self.addr
}

// accessor for KAD self count
func (self *Kademlia) Count() int {
	return self.count
}

// Start brings up a pool of entries potentially from an offline persisted source
// and sets default values for optional parameters
func (self *Kademlia) Start() error {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.quitC != nil {
		return nil
	}
	if self.MaxProx == 0 {
		self.MaxProx = maxProx
	}
	if self.BucketSize == 0 {
		self.BucketSize = bucketSize
	}
	// runtime parameters
	if self.MaxProxBinSize == 0 {
		self.MaxProxBinSize = self.BucketSize
	}

	self.buckets = make([]*bucket, self.MaxProx+1)
	for i, _ := range self.buckets {
		self.buckets[i] = &bucket{size: self.BucketSize} // will initialise bucket{int(0),[]Node(nil),sync.Mutex}
	}

	self.nodeDB = make([][]*nodeRecord, 8*len(self.addr))
	self.nodeIndex = make(map[Address]*nodeRecord)

	self.quitC = make(chan bool)
	return nil
}

// Stop saves the routing table into a persistant form
func (self *Kademlia) Stop(path string) (err error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.quitC == nil {
		return
	}
	close(self.quitC)
	self.quitC = nil

	if len(path) > 0 {
		err = self.Save(path)
		if err != nil {
			kadlogger.Warnf("unable to save node records: %v", err)
		}
	}
	return
}

// RemoveNode is the entrypoint where nodes are taken offline
func (self *Kademlia) RemoveNode(node Node) (err error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	index := self.proximityBin(node.Addr())
	bucket := self.buckets[index]
	for i := 0; i < len(bucket.nodes); i++ {
		if node.Addr() == bucket.nodes[i].Addr() {
			bucket.nodes = append(bucket.nodes[:i], bucket.nodes[(i+1):]...)
		}
	}
	self.count--
	if len(bucket.nodes) < bucket.size {
		err = fmt.Errorf("insufficient nodes (%v) in bucket %v", len(bucket.nodes), index)
	}
	if len(bucket.nodes) == 0 {
		self.adjustProx(index, -1)
	}
	// async callback to notify user that bucket needs filling
	// action is left to the user
	if self.GetNode != nil {
		go self.GetNode(index)
	}
	return
}

// AddNode is the entry point where new nodes are registered
func (self *Kademlia) AddNode(node Node) (err error) {

	self.lock.Lock()
	defer self.lock.Unlock()

	index := self.proximityBin(node.Addr())
	kadlogger.Debugf("bin %d, len: %d\n", index, len(self.buckets))

	bucket := self.buckets[index]
	err = bucket.insert(node)
	if err != nil {
		return
	}
	self.count++
	if index >= self.proxLimit {
		self.adjustProx(index, 1)
	}

	go func() {
		record, found := self.nodeIndex[node.Addr()]
		if found {
			record.node = node
		} else {
			record = &nodeRecord{
				Address: node.Addr(),
				// Url:     node.Url(),
				Active: node.LastActive().UnixNano(),
				node:   node,
			}
			self.nodeIndex[node.Addr()] = record
			self.nodeDB[index] = append(self.nodeDB[index], record)
		}
	}()

	kadlogger.Infof("add peer %v...", node)
	return

}

// adjust Prox (proxLimit and proxSize after an insertion of add nodes into bucket r)
func (self *Kademlia) adjustProx(r int, add int) {
	switch {
	case add > 0 && r == self.proxLimit:
		self.proxLimit += add
		for ; self.proxLimit < self.MaxProx && len(self.buckets[self.proxLimit].nodes) > 0; self.proxLimit++ {
			self.proxSize -= len(self.buckets[self.proxLimit].nodes)
		}
	case add > 0 && r > self.proxLimit && self.proxSize+add > self.MaxProxBinSize:
		self.proxLimit++
		self.proxSize -= len(self.buckets[r].nodes) - add
	case add > 0 && r > self.proxLimit:
		self.proxSize += add
	case add < 0 && r < self.proxLimit && len(self.buckets[r].nodes) == 0:
		for i := self.proxLimit - 1; i > r; i-- {
			self.proxSize += len(self.buckets[i].nodes)
		}
		self.proxLimit = r
	}
}

/*
GetNodes(target) returns the list of nodes belonging to the same proximity bin
as the target. The most proximate bin will be the union of the bins between
proxLimit and MaxProx. proxLimit is dynamically adjusted so that 1) there is no
empty buckets in bin < proxLimit and 2) the sum of all items are the maximum
possible but lower than MaxProxBinSize
*/
func (self *Kademlia) GetNodes(target Address, max int) (r nodesByDistance) {
	self.lock.RLock()
	defer self.lock.RUnlock()
	r.target = target
	index := self.proximityBin(target)
	start := index
	var down bool
	if index >= self.proxLimit {
		start = self.MaxProx
		down = true
	}
	var n int
	limit := max
	if max == 0 {
		limit = 1000
	}
	for {
		bucket := self.buckets[start].nodes
		for i := 0; i < len(bucket); i++ {
			r.push(bucket[i], limit)
			n++
		}
		if max == 0 && start == index ||
			max > 0 && down && start <= index && (n >= max || n == self.Count() || start == 0) {
			break
		}
		if down {
			start--
		} else {
			if start == self.MaxProx {
				if index == 0 {
					break
				}
				start = index - 1
				down = true
			} else {
				start++
			}
		}
	}
	return
}

// in situ mutable bucket
type bucket struct {
	size  int
	nodes []Node
	lock  sync.RWMutex
}

func (a Address) Bin() string {
	var bs []string
	for _, b := range a[:] {
		bs = append(bs, fmt.Sprintf("%08b", b))
	}
	return strings.Join(bs, "")
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
			if proxCmp(target, node.Addr(), last) < 0 {
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
		return proxCmp(h.target, h.nodes[i].Addr(), node.Addr()) >= 0
	})

	if len(h.nodes) < max {
		h.nodes = append(h.nodes, node)
	}
	if ix < len(h.nodes) {
		copy(h.nodes[ix+1:], h.nodes[ix:])
		h.nodes[ix] = node
	}
}

// insert adds a peer to a bucket either by appending to existing items if
// bucket length does not exceed bucketLength, or by replacing the worst
// Node in the bucket
func (self *bucket) insert(node Node) (err error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	if len(self.nodes) >= self.size { // >= allows us to add peers beyond the bucketsize limitation
		worst := self.worstNode()
		self.nodes[worst] = node
	} else {
		self.nodes = append(self.nodes, node)
	}
	return
}

// worst expunges the single worst entry in a row, where worst entry is with a peer that has not been active the longests
func (self *bucket) worstNode() (index int) {
	var oldest time.Time
	for i, node := range self.nodes {
		if (oldest == time.Time{}) || node.LastActive().Before(oldest) {
			oldest = node.LastActive()
			index = i
		}
	}
	return
}

/*
Taking the proximity value relative to a fix point x classifies the points in
the space (n byte long byte sequences) into bins the items in which are each at
most half as distant from x as items in the previous bin. Given a sample of
uniformly distrbuted items (a hash function over arbitrary sequence) the
proximity scale maps onto series of subsets with cardinalities on a negative
exponential scale.

It also has the property that any two item belonging to the same bin are at
most half as distant from each other as they are from x.

If we think of random sample of items in the bins as connections in a network of interconnected nodes than relative proximity can serve as the basis for local
decisions for graph traversal where the task is to find a route between two
points. Since in every step of forwarding, the finite distance halves, there is
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

/*
The distance metric MSB(x, y) of two equal length byte sequences x an y is the
value of the binary integer cast of the xor-ed byte sequence (most significant
bit first).
proximity(x, y) counts the common zeros in the front of this distance measure.
*/
func proximity(one, other Address) (ret int) {
	for i := 0; i < len(one); i++ {
		oxo := one[i] ^ other[i]
		for j := 0; j < 8; j++ {
			if (oxo>>uint8(7-j))&0x1 != 0 {
				return i*8 + j
			}
		}
	}
	return len(one)*8 - 1
}

// proxCmp compares the distances a->target and b->target.
// Returns -1 if a is closer to target, 1 if b is closer to target
// and 0 if they are equal.
func proxCmp(target, a, b Address) int {
	for i := range target {
		da := a[i] ^ target[i]
		db := b[i] ^ target[i]
		if da > db {
			return 1
		} else if da < db {
			return -1
		}
	}
	return 0
}

func (self *Kademlia) DB() [][]*nodeRecord {
	return self.nodeDB
}

func (n *nodeRecord) bumpActive() {
	stamp := time.Now().Unix()
	atomic.StoreInt64(&n.Active, stamp)
}

func (n *nodeRecord) LastActive() time.Time {
	stamp := atomic.LoadInt64(&n.Active)
	return time.Unix(stamp, 0)
}

// save persists all peers encountered
func (self *Kademlia) Save(path string) error {

	kad := kadDB{
		Address: self.addr,
		Nodes:   self.nodeDB,
	}
	for _, b := range kad.Nodes {
		for _, node := range b {
			node.setActive()
		}
	}
	data, err := json.MarshalIndent(&kad, "", " ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, os.ModePerm)
}

// loading the idle node record from disk
func (self *Kademlia) Load(path string) (err error) {
	var data []byte
	data, err = ioutil.ReadFile(path)
	if err != nil {
		return
	}
	var kad kadDB
	err = json.Unmarshal(data, &kad)
	if err != nil {
		return
	}
	self.nodeDB = kad.Nodes
	if self.addr != kad.Address {
		return fmt.Errorf("invalid kad db: address mismatch, expected %v, got %v", self.addr, kad.Address)
	}
	return
}
