// Copyright 2017 The go-ethereum Authors
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
package network

import (
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/pot"
)

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

proxLimit is dynamically adjusted so that
1) there is no empty buckets in bin < proxLimit and
2) the sum of all items are the minimum possible but higher than ProxBinSize
adjust Prox (proxLimit and proxSize after an insertion/removal of nodes)
caller holds the lock

*/

// default values for Kademlia Parameters
const (
	bucketSize   = 4
	proxBinSize  = 2
	maxProx      = 8
	connRetryExp = 2
)

var (
	purgeInterval        = 42 * time.Hour
	initialRetryInterval = 42 * time.Millisecond
	maxIdleInterval      = 42 * 1000 * time.Millisecond
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

// NewKadParams() returns a params struct with default values
func NewKadParams() *KadParams {
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

// Kademlia is a table of live peers and a db of known peers
type Kademlia struct {
	addr         *pot.HashAddress // immutable baseaddress of the table
	*KadParams                    // Kademlia configuration parameters
	conns, peers *pot.Pot         // pots container for peers
}

// NewKademlia(addr, params) creates a Kademlia table for base address addr
// with parameters as in params
// if params is nil, it uses default values
func NewKademlia(addr []byte, params *KadParams) *Kademlia {
	if params == nil {
		params = NewKadParams()
	}
	base := pot.NewHashAddressFromBytes(addr)
	return &Kademlia{
		addr:      base,
		KadParams: params,
		conns:     pot.NewPot(nil, 0),
		peers:     pot.NewPot(nil, 0),
	}
}

// KadPeer represents a Kademlia Peer and extends
// * PeerAddr interface (overlay and underlay addresses)
// * Peer interface (id, last seen, drop)
// * HashAddress as derived from PeerAddr overlay implement pot.PoVal interface
type KadPeer struct {
	*pot.HashAddress
	PeerAddr
	Peer            Peer
	seenAt, readyAt time.Time
}

func (self *KadPeer) String() string {
	return string(self.OverlayAddr())
}

func (self *Kademlia) callable(kp *KadPeer) bool {
	if kp.Peer != nil || kp.readyAt.After(time.Now()) {
		return false
	}
	delta := time.Since(time.Time(kp.seenAt))
	if delta < self.InitialRetryInterval {
		delta = self.InitialRetryInterval
	}
	interval := time.Duration(delta * time.Duration(self.ConnRetryExp))

	glog.V(logger.Detail).Infof("peer %v ready to be tried. seen at %v (%v ago), scheduled at %v", kp, kp.seenAt, delta, kp.readyAt)

	// scheduling next check
	kp.readyAt = time.Now().Add(interval)
	return true
}

// NewKadPeer(na) creates a kademlia peer from a PeerAddr interface
func NewKadPeer(na PeerAddr) *KadPeer {
	return &KadPeer{
		HashAddress: pot.NewHashAddressFromBytes(na.OverlayAddr()),
		PeerAddr:    na,
	}
}

// Register(nas) enters each PeerAddr as kademlia peers into the
// database of known peers
func (self *Kademlia) Register(nas ...PeerAddr) error {
	np := pot.NewPot(nil, 0)
	for _, na := range nas {
		p := NewKadPeer(na)
		np, _, _ = pot.Add(np, pot.PotVal(p))
	}
	common := self.peers.Merge(np)
	glog.V(6).Infof("add peers: %v out of %v new", np.Size()-common, np.Size())
	return nil
}

// On(p) inserts the peer as a kademlia peer into the live peers
func (self *Kademlia) On(p Peer) {
	kp := NewKadPeer(p)
	self.conns.Swap(kp, func(v pot.PotVal) pot.PotVal {
		if v == nil {
			self.peers.Swap(kp, func(v pot.PotVal) pot.PotVal {
				if v != nil {
					kp = v.(*KadPeer)
				}
				kp.Peer = p
				return pot.PotVal(kp)
			})
			return pot.PotVal(kp)
		}
		return v
	})
}

// Off removes a peer from among live peers
func (self *Kademlia) Off(p Peer) {
	kp := NewKadPeer(p)
	self.conns.Remove(kp)
	kp.Peer = nil
}

// EachLivePeer(base, po, f) is an iterator applying f to each live peer
// that has proximity order po as measure from the base
func (self *Kademlia) EachLivePeer(base []byte, o int, f func(Peer) bool) {
	var p pot.PotVal
	if base == nil {
		p = pot.PotVal(self.addr)
	} else {
		p = pot.NewHashAddressFromBytes(base)
	}
	self.conns.EachNeighbour(p, func(val pot.PotVal, po int) bool {
		if po == o {
			return f(val.(*KadPeer).Peer)
		}
		return po < o
	})
}

// EachPeer(base, po, f) is an iterator applying f to each known peer
// that has proximity order po as measure from the base
func (self *Kademlia) EachPeer(base []byte, o int, f func(PeerAddr) bool) {
	var p pot.PotVal
	if base == nil {
		p = pot.PotVal(self.addr)
	} else {
		p = pot.NewHashAddressFromBytes(base)
	}
	self.peers.EachNeighbour(p, func(val pot.PotVal, po int) bool {
		if po == o {
			return f(val.(*KadPeer).PeerAddr)
		}
		return po < o
	})
}

// proxLimit() returns the proximity order that defines the distance of
// the nearest neighbour set with cardinality >= ProxBinSize
func (self *Kademlia) proxLimit() int {
	var proxLimit int
	var size int
	f := func(v pot.PotVal, i int) bool {
		size++
		proxLimit = i
		return size <= self.BucketSize
	}
	self.conns.EachNeighbour(pot.PotVal(self.addr), f)
	return proxLimit
}

// SuggestPeer returns a known peer for the lowest proximity bin for the
// lowest bincount below proxLimit
// naturally if there is an empty row it returns a peer for that
//
func (self *Kademlia) SuggestPeer() (p PeerAddr, o int, want bool) {
	minsize := self.BucketSize
	proxLimit := self.proxLimit()
	var bpo []int
	self.conns.EachBin(0, func(po, size int, f func(func(val pot.PotVal, i int) bool) bool) bool {
		if size < minsize {
			minsize = size
			bpo = append(bpo, po)
		}
		return size > 0 && po < proxLimit
	})
	// all buckets are full
	// minsize == self.BucketSize
	if len(bpo) == 0 {
		return nil, 0, false
	}
	// as long as we got candidate peers to connect to
	// just ask for closest peers
	o = 256
	// try to select a peer
	for i := len(bpo) - 1; i >= 0; i-- {
		// find the first callable peer
		self.peers.EachBin(bpo[i], func(po, size int, f func(func(val pot.PotVal, i int) bool) bool) bool {
			f(func(val pot.PotVal, i int) bool {
				cp := val.(*KadPeer)
				if self.callable(cp) {
					p = cp
					return false
				}
				return true
			})
			return false
		})
		// found a candidate
		if p != nil {
			break
		}
		// cannot find a candidate, ask for more for this proximity bin specifically
		o = bpo[i]
	}
	return p, o, true
}

// kademlia table + kaddb table displayed with ascii
func (self *Kademlia) String() string {

	var rows []string

	rows = append(rows, "=========================================================================")
	rows = append(rows, fmt.Sprintf("%v KΛÐΞMLIΛ hive: queen's address: %v", time.Now().UTC().Format(time.UnixDate), self.addr.String()[:6]))
	rows = append(rows, fmt.Sprintf("population: %d (%d), ProxBinSize: %d, BucketSize: %d", self.conns.Size(), self.peers.Size(), self.ProxBinSize, self.BucketSize))

	liverows := make([]string, self.MaxProx)
	peersrows := make([]string, self.MaxProx)
	var proxLimit int
	rest := self.conns.Size()
	self.conns.EachBin(0, func(po, size int, f func(func(val pot.PotVal, i int) bool) bool) bool {
		var rowlen int
		row := []string{fmt.Sprintf("%2d", size)}
		rest -= size
		f(func(val pot.PotVal, vpo int) bool {
			row = append(row, val.(*KadPeer).String()[:6])
			rowlen++
			return rowlen < 4
		})
		if rowlen == 0 || rest < self.ProxBinSize {
			proxLimit = po - 1
		}
		for ; rowlen < 4; rowlen++ {
			row = append(row, "      ")
		}
		liverows[po] = strings.Join(row, " ")
		return true
	})

	self.peers.EachBin(0, func(po, size int, f func(func(val pot.PotVal, i int) bool) bool) bool {
		var rowlen int
		row := []string{fmt.Sprintf("| %2d", size)}
		f(func(val pot.PotVal, vpo int) bool {
			kp := val.(*KadPeer)
			if kp.Peer != nil {
				return true
			}
			row = append(row, kp.String()[:6])
			rowlen++
			return rowlen < 4
		})
		peersrows[po] = strings.Join(row, " ")
		return true
	})

	for i := 0; i < self.MaxProx; i++ {
		if i == proxLimit {
			rows = append(rows, fmt.Sprintf("============ PROX LIMIT: %d ==========================================", i))
		}
		left := liverows[i]
		right := peersrows[i]
		if len(left) == 0 {
			left = "0                                           "
		}
		if len(right) == 0 {
			right = "0                                           "
		}
		rows = append(rows, fmt.Sprintf("%03d %v | %v", i, left, right))
	}
	rows = append(rows, "=========================================================================")
	return strings.Join(rows, "\n")
}
