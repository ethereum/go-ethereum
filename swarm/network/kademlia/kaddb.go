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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

type NodeData interface {
	json.Marshaler
	json.Unmarshaler
}

// allow inactive peers under
type NodeRecord struct {
	Addr  Address          // address of node
	Url   string           // Url, used to connect to node
	After time.Time        // next call after time
	Seen  time.Time        // last connected at time
	Meta  *json.RawMessage // arbitrary metadata saved for a peer

	node Node
}

func (self *NodeRecord) setSeen() {
	t := time.Now()
	self.Seen = t
	self.After = t
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
	lock                 sync.RWMutex
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
		glog.V(logger.Info).Infof("add new record %v to kaddb", record)
		// insert in kaddb
		self.index[a] = record
		self.Nodes[index] = append(self.Nodes[index], record)
	} else {
		glog.V(logger.Info).Infof("found record %v in kaddb", record)
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
			glog.V(logger.Detail).Infof("new nodes: %v (keys: %v)\nnodes: %v", newnodes, nodes)
			self.Nodes[index] = newnodes
			n++
		}
	}
	if n > 0 {
		glog.V(logger.Debug).Infof("%d/%d node records (new/known)", n, len(nrs))
	}
}

/*
next return one node record with the highest priority for desired
connection.
This is used to pick candidates for live nodes that are most wanted for
a higly connected low centrality network structure for Swarm which best suits
for a Kademlia-style routing.

* Starting as naive node with empty db, this implements Kademlia bootstrapping
* As a mature node, it fills short lines. All on demand.

The candidate is chosen using the following strategy:
We check for missing online nodes in the buckets for 1 upto Max BucketSize rounds.
On each round we proceed from the low to high proximity order buckets.
If the number of active nodes (=connected peers) is < rounds, then start looking
for a known candidate. To determine if there is a candidate to recommend the
kaddb node record database row corresponding to the bucket is checked.

If the row cursor is on position i, the ith element in the row is chosen.
If the record is scheduled not to be retried before NOW, the next element is taken.
If the record is scheduled to be retried, it is set as checked, scheduled for
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


The second argument returned names the first missing slot found
*/
func (self *KadDb) findBest(maxBinSize int, binSize func(int) int) (node *NodeRecord, need bool, proxLimit int) {
	// return nil, proxLimit indicates that all buckets are filled
	defer self.lock.Unlock()
	self.lock.Lock()

	var interval time.Duration
	var found bool
	var purge []bool
	var delta time.Duration
	var cursor int
	var count int
	var after time.Time

	// iterate over columns maximum bucketsize times
	for rounds := 1; rounds <= maxBinSize; rounds++ {
	ROUND:
		// iterate over rows from PO 0 upto MaxProx
		for po, dbrow := range self.Nodes {
			// if row has rounds connected peers, then take the next
			if binSize(po) >= rounds {
				continue ROUND
			}
			if !need {
				// set proxlimit to the PO where the first missing slot is found
				proxLimit = po
				need = true
			}
			purge = make([]bool, len(dbrow))

			// there is a missing slot - finding a node to connect to
			// select a node record from the relavant kaddb row (of identical prox order)
		ROW:
			for cursor = self.cursors[po]; !found && count < len(dbrow); cursor = (cursor + 1) % len(dbrow) {
				count++
				node = dbrow[cursor]

				// skip already connected nodes
				if node.node != nil {
					glog.V(logger.Debug).Infof("kaddb record %v (PO%03d:%d/%d) already connected", node.Addr, po, cursor, len(dbrow))
					continue ROW
				}

				// if node is scheduled to connect
				if time.Time(node.After).After(time.Now()) {
					glog.V(logger.Debug).Infof("kaddb record %v (PO%03d:%d) skipped. seen at %v (%v ago), scheduled at %v", node.Addr, po, cursor, node.Seen, delta, node.After)
					continue ROW
				}

				delta = time.Since(time.Time(node.Seen))
				if delta < self.initialRetryInterval {
					delta = self.initialRetryInterval
				}
				if delta > self.purgeInterval {
					// remove node
					purge[cursor] = true
					glog.V(logger.Debug).Infof("kaddb record %v (PO%03d:%d) unreachable since %v. Removed", node.Addr, po, cursor, node.Seen)
					continue ROW
				}

				glog.V(logger.Debug).Infof("kaddb record %v (PO%03d:%d) ready to be tried. seen at %v (%v ago), scheduled at %v", node.Addr, po, cursor, node.Seen, delta, node.After)

				// scheduling next check
				interval = time.Duration(delta * time.Duration(self.connRetryExp))
				after = time.Now().Add(interval)

				glog.V(logger.Debug).Infof("kaddb record %v (PO%03d:%d) selected as candidate connection %v. seen at %v (%v ago), selectable since %v, retry after %v (in %v)", node.Addr, po, cursor, rounds, node.Seen, delta, node.After, after, interval)
				node.After = after
				found = true
			} // ROW
			self.cursors[po] = cursor
			self.delete(po, purge)
			if found {
				return node, need, proxLimit
			}
		} // ROUND
	} // ROUNDS

	return nil, need, proxLimit
}

// deletes the noderecords of a kaddb row corresponding to the indexes
// caller must hold the dblock
// the call is unsafe, no index checks
func (self *KadDb) delete(row int, purge []bool) {
	var nodes []*NodeRecord
	dbrow := self.Nodes[row]
	for i, del := range purge {
		if i == self.cursors[row] {
			//reset cursor
			self.cursors[row] = len(nodes)
		}
		// delete the entry to be purged
		if del {
			delete(self.index, dbrow[i].Addr)
			continue
		}
		// otherwise append to new list
		nodes = append(nodes, dbrow[i])
	}
	self.Nodes[row] = nodes
}

// save persists kaddb on disk (written to file on path in json format.
func (self *KadDb) save(path string, cb func(*NodeRecord, Node)) error {
	defer self.lock.Unlock()
	self.lock.Lock()

	var n int

	for _, b := range self.Nodes {
		for _, node := range b {
			n++
			node.After = time.Now()
			node.Seen = time.Now()
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
		glog.V(logger.Warn).Infof("unable to save kaddb with %v nodes to %v: err", n, path, err)
	} else {
		glog.V(logger.Info).Infof("saved kaddb with %v nodes to %v", n, path)
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
	var purge []bool
	for po, b := range self.Nodes {
		purge = make([]bool, len(b))
	ROW:
		for i, node := range b {
			if cb != nil {
				err = cb(node, node.node)
				if err != nil {
					purge[i] = true
					continue ROW
				}
			}
			n++
			if (node.After == time.Time{}) {
				node.After = time.Now()
			}
			self.index[node.Addr] = node
		}
		self.delete(po, purge)
	}
	glog.V(logger.Info).Infof("loaded kaddb with %v nodes from %v", n, path)

	return
}

// accessor for KAD offline db count
func (self *KadDb) count() int {
	defer self.lock.Unlock()
	self.lock.Lock()
	return len(self.index)
}
