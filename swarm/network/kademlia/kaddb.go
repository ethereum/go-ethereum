package kademlia

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

type Time time.Time

func (t *Time) MarshalJSON() (out []byte, err error) {
	return []byte(fmt.Sprintf("%d", t.Unix())), nil
}

func (t *Time) UnmarshalJSON(value []byte) error {
	var i int64
	_, err := fmt.Sscanf(string(value), "%d", &i)
	if err != nil {
		return err
	}
	*t = Time(time.Unix(i, 0))
	return nil
}

func (t Time) Unix() int64 {
	return time.Time(t).Unix()
}

type NodeData interface {
	json.Marshaler
	json.Unmarshaler
}

// allow inactive peers under
type NodeRecord struct {
	Addr  Address          // address of node
	Url   string           // Url, used to connect to node
	After Time             // next call after time
	Seen  Time             // last connected at time
	Meta  *json.RawMessage // arbitrary metadata saved for a peer

	node      Node
	connected bool
}

// set checked to current time,
func (self *NodeRecord) setSeen() {
	self.Seen = Time(time.Now())
}

func (self *NodeRecord) String() string {
	return fmt.Sprintf("<%v>", self.Addr)
}

// persisted node record database ()
type KadDb struct {
	Address              Address
	Nodes                [][]*NodeRecord
	index                map[Address]*NodeRecord
	cursors              []int
	lock                 sync.Mutex
	purgeInterval        time.Duration
	initialRetryInterval time.Duration
	connRetryExp         int
}

func newKadDb(addr Address, params *KadParams) *KadDb {
	return &KadDb{
		Address:              addr,
		Nodes:                make([][]*NodeRecord, params.MaxProx+1), // overwritten by load
		cursors:              make([]int, params.MaxProx+1),
		index:                make(map[Address]*NodeRecord),
		purgeInterval:        params.PurgeInterval,
		initialRetryInterval: params.InitialRetryInterval,
		connRetryExp:         params.ConnRetryExp,
	}
}

func (self *KadDb) findOrCreate(index int, a Address, url string) *NodeRecord {
	defer self.lock.Unlock()
	self.lock.Lock()

	record, found := self.index[a]
	if !found {
		record = &NodeRecord{
			Addr: a,
			Url:  url,
		}
		glog.V(logger.Info).Infof("[KΛÐ]: add new record %v to kaddb", record)
		// insert in kaddb
		self.index[a] = record
		self.Nodes[index] = append(self.Nodes[index], record)
	} else {
		glog.V(logger.Info).Infof("[KΛÐ]: found record %v in kaddb", record)
	}
	// update last seen time
	record.setSeen()
	// update with url in case IP/port changes
	record.Url = url
	return record
}

// add adds node records to kaddb (persisted node record db)
func (self *KadDb) add(nrs []*NodeRecord, proximityBin func(Address) int) {
	defer self.lock.Unlock()
	self.lock.Lock()
	var n int
	var nodes []*NodeRecord
	for _, node := range nrs {
		_, found := self.index[node.Addr]
		if !found && node.Addr != self.Address {
			node.setSeen()
			self.index[node.Addr] = node
			index := proximityBin(node.Addr)
			dbcursor := self.cursors[index]
			nodes = self.Nodes[index]
			// this is inefficient for allocation, need to just append then shift
			newnodes := make([]*NodeRecord, len(nodes)+1)
			copy(newnodes[:], nodes[:dbcursor])
			newnodes[dbcursor] = node
			copy(newnodes[dbcursor+1:], nodes[dbcursor:])
			glog.V(logger.Detail).Infof("[KΛÐ]: new nodes: %v (keys: %v)\nnodes: %v", newnodes, nodes)
			self.Nodes[index] = newnodes
			n++
		}
	}
	if n > 0 {
		glog.V(logger.Debug).Infof("[KΛÐ]: %d/%d node records (new/known)", n, len(nrs))
	}
}

/*
next return one node record with the highest priority for desired
connection.
This is used to pick candidates for live nodes that are most wanted for
a higly connected low centrality network structure for Swarm which best suits
for a Kademlia-style routing.

The candidate is chosen using the following strategy.
We check for missing online nodes in the buckets for 1 upto Max BucketSize rounds.
On each round we proceed from the low to high proximity order buckets.
If the number of active nodes (=connected peers) is < rounds, then start looking
for a known candidate. To determine if there is a candidate to recommend the
node record database row corresponding to the bucket is checked.

If the row cursor is on position i, the ith element in the row is chosen.
If the record is scheduled not to be retried before NOW, the next element is taken.
If the record is scheduled can be retried, it is set as checked, scheduled for
checking and is returned. The time of the next check is in X (duration) such that
X = ConnRetryExp * delta where delta is the time past since the last check and
ConnRetryExp is constant obsoletion factor. (Note that when node records are added
from peer messages, they are marked as checked and placed at the cursor, ie.
given priority over older entries). Entries which were checked more than
purgeInterval ago are deleted from the kaddb row. If no candidate is found after
a full round of checking the next bucket up is considered. If no candidate is
found when we reach the maximum-proximity bucket, the next round starts.

node record a is more favoured to b a > b iff a is a passive node (record of
offline past peer)
|proxBin(a)| < |proxBin(b)|
|| (proxBin(a) < proxBin(b) && |proxBin(a)| == |proxBin(b)|)
|| (proxBin(a) == proxBin(b) && lastChecked(a) < lastChecked(b))

This has double role. Starting as naive node with empty db, this implements
Kademlia bootstrapping
As a mature node, it fills short lines. All on demand.

The second argument returned names the first missing slot found
*/
func (self *KadDb) findBest(bucketSize int, binsize func(int) int) (node *NodeRecord, proxLimit int) {
	// return value -1 indicates that buckets are filled in all
	proxLimit = -1
	defer self.lock.Unlock()
	self.lock.Lock()

	var interval int64
	var found bool
	for rounds := 1; rounds <= bucketSize; rounds++ {
	ROUND:
		for po, dbrow := range self.Nodes {
			if po > len(self.Nodes) {
				break ROUND
			}
			size := binsize(po)
			if size < rounds {
				if proxLimit < 0 {
					// set the first missing slot found
					proxLimit = po
				}
				var count int
				var purge []int
				n := self.cursors[po]

				// try node records in the relavant kaddb row (of identical prox order)
				// if they are ripe for checking
			ROW:
				for count < len(dbrow) {
					node = dbrow[n]

					// skip already connected nodes
					if !node.connected {

						glog.V(logger.Detail).Infof("[KΛÐ]: kaddb record %v (PO%03d:%d) not to be retried before %v", node.Addr, po, n, node.After)

						// time since last known connection attempt
						delta := node.After.Unix() - node.Seen.Unix()
						// if delta < 4 {
						// 	node.After = Time(time.Time{})
						// }

						// if node is scheduled to connect
						if time.Time(node.After).Before(time.Now()) {

							// if checked longer than purge interval
							if time.Time(node.Seen).Add(self.purgeInterval).Before(time.Now()) {
								// delete node
								purge = append(purge, n)
								glog.V(logger.Detail).Infof("[KΛÐ]: inactive node record %v (PO%03d:%d) last check: %v, next check: %v", node.Addr, po, n, node.Seen, node.After)
							} else {
								// scheduling next check
								if (node.After == Time(time.Time{})) {
									node.After = Time(time.Now().Add(self.initialRetryInterval))
								} else {
									interval = delta * int64(self.connRetryExp)
									node.After = Time(time.Unix(time.Now().Unix()+interval, 0))
								}

								glog.V(logger.Detail).Infof("[KΛÐ]: serve node record %v (PO%03d:%d), last check: %v,  next check: %v", node.Addr, po, n, node.Seen, node.After)
							}
							found = true
							break ROW
						}
						glog.V(logger.Detail).Infof("[KΛÐ]: kaddb record %v (PO%03d:%d) not ready. skipped. not to be retried before: %v", node.Addr, po, n, node.After)
					} // if node.node == nil
					n++
					count++
					// cycle: n = n %  len(dbrow)
					if n >= len(dbrow) {
						n = 0
					}
				}
				self.cursors[po] = n
				self.delete(po, purge...)
				if found {
					glog.V(logger.Detail).Infof("[KΛÐ]: rounds %d: prox limit: PO%03d\n%v", rounds, proxLimit, node)
					node.setSeen()
					return
				}
			} // if len < rounds
		} // for po-s
		glog.V(logger.Detail).Infof("[KΛÐ]: rounds %d: proxlimit: PO%03d", rounds, proxLimit)
		if proxLimit == 0 || proxLimit < 0 && bucketSize == rounds {
			return
		}
	} // for round

	return
}

// deletes the noderecords of a kaddb row corresponding to the indexes
// caller must hold the dblock
// the call is unsafe, no index checks
func (self *KadDb) delete(row int, indexes ...int) {
	var prev int
	var nodes []*NodeRecord
	dbrow := self.Nodes[row]
	for _, next := range indexes {
		// need to adjust dbcursor
		if next > 0 {
			if next <= self.cursors[row] {
				self.cursors[row]--
			}
			nodes = append(nodes, dbrow[prev:next]...)
		}
		prev = next + 1
		delete(self.index, dbrow[next].Addr)
	}
	self.Nodes[row] = append(nodes, dbrow[prev:]...)
}

// save persists kaddb on disk (written to file on path in json format.
func (self *KadDb) save(path string, cb func(*NodeRecord, Node)) error {
	defer self.lock.Unlock()
	self.lock.Lock()

	var n int

	for _, b := range self.Nodes {
		for _, node := range b {
			n++
			node.After = Time(time.Now())
			node.Seen = Time(time.Now())
			if cb != nil {
				cb(node, node.node)
			}
		}
	}

	data, err := json.MarshalIndent(self, "", " ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, data, os.ModePerm)
	if err != nil {
		glog.V(logger.Warn).Infof("[KΛÐ]: unable to save kaddb with %v nodes to %v: err", n, path, err)
	} else {
		glog.V(logger.Info).Infof("[KΛÐ] saved kaddb with %v nodes to %v", n, path)
	}
	return err
}

// Load(path) loads the node record database (kaddb) from file on path.
func (self *KadDb) load(path string, cb func(*NodeRecord, Node) error) (err error) {
	defer self.lock.Unlock()
	self.lock.Lock()

	var data []byte
	data, err = ioutil.ReadFile(path)
	if err != nil {
		return
	}

	err = json.Unmarshal(data, self)
	if err != nil {
		return
	}
	var n int
	var purge []int
	for po, b := range self.Nodes {
	ROW:
		for i, node := range b {
			if cb != nil {
				err = cb(node, node.node)
				if err != nil {
					purge = append(purge, i)
					continue ROW
				}
			}
			n++
			if (node.After == Time(time.Time{})) {
				node.After = Time(time.Now())
			}
			self.index[node.Addr] = node
		}
		self.delete(po, purge...)
	}
	glog.V(logger.Info).Infof("[KΛÐ] loaded kaddb with %v nodes from %v", n, path)

	return
}

// accessor for KAD offline db count
func (self *KadDb) count() int {
	defer self.lock.Unlock()
	self.lock.Lock()
	return len(self.index)
}
