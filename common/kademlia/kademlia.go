package kademlia

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

const (
	bucketSize   = 20
	maxProx      = 255
	connRetryExp = 2
)

var (
	purgeInterval        = 42 * time.Hour
	initialRetryInterval = 42 * 100 * time.Millisecond
)

type KadParams struct {
	// adjustable parameters
	MaxProx              int
	ProxBinSize          int
	BucketSize           int
	PurgeInterval        time.Duration
	InitialRetryInterval time.Duration
	ConnRetryExp         int
}

func NewKadParams() *KadParams {
	return &KadParams{
		MaxProx:              maxProx,
		ProxBinSize:          bucketSize,
		BucketSize:           bucketSize,
		PurgeInterval:        purgeInterval,
		InitialRetryInterval: initialRetryInterval,
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
	buckets    []*bucket    // the actual bins
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
	buckets := make([]*bucket, params.MaxProx+1)
	for i, _ := range buckets {
		buckets[i] = &bucket{size: params.BucketSize} // will initialise bucket{int(0),[]Node(nil),sync.Mutex}
	}
	glog.V(logger.Info).Infof("[KΛÐ] base address %v", addr)

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
	index := self.proximityBin(node.Addr())
	record := self.db.findOrCreate(index, node.Addr(), node.Url())
	// callback on add node
	// setting the node on the record, set it checked (for connectivity)
	record.node = node

	glog.V(logger.Info).Infof("[KΛÐ]: add node record %v with node %v", record, node)
	if cb != nil {
		err = cb(record, node)
		glog.V(logger.Info).Infof("[KΛÐ]: cb(%v, %v) ->%v", record, node, err)
		if err != nil {
			return fmt.Errorf("node %v not added: %v", node.Addr(), err)
		}
	}
	record.connected = true

	defer self.lock.Unlock()
	self.lock.Lock()

	// insert in kademlia table of active nodes
	bucket := self.buckets[index]
	// if bucket is full insertion replaces the worst node
	// TODO: give priority to peers with active traffic
	if worst, pos := bucket.insert(node); worst != nil {
		glog.V(logger.Info).Infof("[KΛÐ]: replace node %v (%d) with node %v", worst, pos, node)
		// no prox adjustment needed
		// do not change count
	} else {
		glog.V(logger.Info).Infof("[KΛÐ]: add node %v to table", node)
		self.count++
		self.adjustProxMore(index)
	}

	return

}

//  is the entrypoint called when a node is taken offline
func (self *Kademlia) Off(node Node, cb func(*NodeRecord, Node)) (err error) {
	self.lock.Lock()
	defer self.lock.Unlock()

	var found bool
	index := self.proximityBin(node.Addr())
	bucket := self.buckets[index]
	for i := 0; i < len(bucket.nodes); i++ {
		if node.Addr() == bucket.nodes[i].Addr() {
			found = true
			bucket.nodes = append(bucket.nodes[:i], bucket.nodes[(i+1):]...)
		}
	}

	if !found {
		return
	}
	glog.V(logger.Info).Infof("[KΛÐ]: remove node %v from table", node)

	self.count--
	if len(bucket.nodes) < bucket.size {
		err = fmt.Errorf("insufficient nodes (%v) in bucket %v", len(bucket.nodes), index)
	}
	self.adjustProxLess(index)

	r := self.db.index[node.Addr()]
	// callback on remove
	if cb != nil {
		cb(r, r.node)
	}
	r.node = nil
	r.connected = false

	return
}

// proxLimit is dynamically adjusted so that 1) there is no
// empty buckets in bin < proxLimit and 2) the sum of all items sare the maximum
// possible but lower than ProxBinSize
// adjust Prox (proxLimit and proxSize after an insertion of add nodes into bucket r)
func (self *Kademlia) adjustProxMore(r int) {
	if r >= self.proxLimit {
		exLimit := self.proxLimit
		exSize := self.proxSize
		self.proxSize++

		var i int
		for i = self.proxLimit; i < self.MaxProx && len(self.buckets[i].nodes) > 0 && self.proxSize-len(self.buckets[i].nodes) > self.ProxBinSize; i++ {
			self.proxSize -= len(self.buckets[i].nodes)
		}
		self.proxLimit = i

		glog.V(logger.Detail).Infof("[KΛÐ]: Max Prox Bin: Lower Limit: %v (was %v): Bin Size: %v (was %v)", self.proxLimit, exLimit, self.proxSize, exSize)
	}
}

func (self *Kademlia) adjustProxLess(r int) {
	exLimit := self.proxLimit
	exSize := self.proxSize
	if r >= self.proxLimit {
		self.proxSize--
	}

	if r < self.proxLimit && len(self.buckets[r].nodes) == 0 {
		for i := self.proxLimit - 1; i > r; i-- {
			self.proxSize += len(self.buckets[i].nodes)
		}
		self.proxLimit = r
	} else if self.proxLimit > 0 && r >= self.proxLimit-1 {
		var i int
		for i = self.proxLimit - 1; i > 0 && len(self.buckets[i].nodes)+self.proxSize <= self.ProxBinSize; i-- {
			self.proxSize += len(self.buckets[i].nodes)
		}
		self.proxLimit = i
	}

	if exLimit != self.proxLimit || exSize != self.proxSize {
		glog.V(logger.Detail).Infof("[KΛÐ]: Max Prox Bin: Lower Limit: %v (was %v): Bin Size: %v (was %v)", self.proxLimit, exLimit, self.proxSize, exSize)
	}
}

/*
returns the list of nodes belonging to the same proximity bin
as the target. The most proximate bin will be the union of the bins between
proxLimit and MaxProx.
*/
func (self *Kademlia) FindClosest(target Address, max int) []Node {
	defer self.lock.RUnlock()
	self.lock.RLock()
	r := nodesByDistance{
		target: target,
	}
	index := self.proximityBin(target)

	start := index
	var down bool
	if index >= self.proxLimit {
		index = self.proxLimit
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
		if max == 0 && start <= index && (n > 0 || start == 0) ||
			max > 0 && down && start <= index && (n >= limit || n == self.count || start == 0) {
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
	glog.V(logger.Detail).Infof("[KΛÐ]: serve %d (=<%d) nodes for target lookup %v (PO%d)", n, self.MaxProx, target, index)
	return r.nodes
}

func (self *Kademlia) binsize(p int) int {
	b := self.buckets[p]
	defer b.lock.RUnlock()
	b.lock.RLock()
	return len(b.nodes)
}

func (self *Kademlia) FindBest() (node *NodeRecord, proxLimit int) {
	return self.db.findBest(self.BucketSize, self.binsize)
}

//  adds node records to kaddb (persisted node record db)
func (self *Kademlia) Add(nrs []*NodeRecord) {
	self.db.add(nrs, self.proximityBin)
}

// in situ mutable bucket
type bucket struct {
	size  int
	nodes []Node
	lock  sync.RWMutex
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

// insert adds a peer to a bucket either by appending to existing items if
// bucket length does not exceed bucketSize, or by replacing the worst
// Node in the bucket
func (self *bucket) insert(node Node) (dropped Node, pos int) {
	self.lock.Lock()
	defer self.lock.Unlock()
	if len(self.nodes) >= self.size { // >= allows us to add peers beyond the bucketsize limitation
		dropped, pos = self.worstNode()
		if dropped != nil {
			self.nodes[pos] = node
			glog.V(logger.Info).Infof("[KΛÐ] dropping node %v (%d)", dropped, pos)
			dropped.Drop()
			return
		}
	}
	self.nodes = append(self.nodes, node)
	return
}

func (self *bucket) length(node Node) int {
	self.lock.Lock()
	defer self.lock.Unlock()
	return len(self.nodes)
}

// worst expunges the single worst node in a row, where worst entry is the node
// that has been inactive for the longests time
func (self *bucket) worstNode() (node Node, pos int) {
	var oldest time.Time
	for p, n := range self.nodes {
		if (oldest == time.Time{}) || !oldest.Before(n.LastActive()) {
			oldest = n.LastActive()
			node = n
			pos = p
		}
	}
	return
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

	var rows []string
	rows = append(rows, "=========================================================================")
	rows = append(rows, fmt.Sprintf("%v : MaxProx: %d, ProxBinSize: %d, BucketSize: %d, proxLimit: %d, proxSize: %d", time.Now(), self.MaxProx, self.ProxBinSize, self.BucketSize, self.proxLimit, self.proxSize))

	for i, b := range self.buckets {

		if i == self.proxLimit {
			rows = append(rows, fmt.Sprintf("===================== PROX LIMIT: %d =================================", i))
		}
		row := []string{fmt.Sprintf("%03d", i), fmt.Sprintf("%2d", len(b.nodes))}
		var k int
		c := self.db.cursors[i]
		for ; k < len(b.nodes); k++ {
			p := b.nodes[(c+k)%len(b.nodes)]
			row = append(row, fmt.Sprintf("%s", p.Addr().String()[:8]))
			if k == 3 {
				break
			}
		}
		for ; k < 3; k++ {
			row = append(row, "        ")
		}
		row = append(row, fmt.Sprintf("| %2d %2d", len(self.db.Nodes[i]), self.db.cursors[i]))

		for j, p := range self.db.Nodes[i] {
			row = append(row, fmt.Sprintf("%08x", p.Addr[:4]))
			if j == 2 {
				break
			}
		}
		rows = append(rows, strings.Join(row, " "))
		if i == self.MaxProx {
			break
		}
	}
	rows = append(rows, "=========================================================================")
	return strings.Join(rows, "\n")
}
