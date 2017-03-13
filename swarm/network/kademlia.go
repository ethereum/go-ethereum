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
type KadDiscovery interface {
	NotifyPeer(Peer, uint8) error
	NotifyProx(uint8) error
}

type KadParams struct {
	// adjustable parameters
	MaxProxDisplay int
	MinProxBinSize int
	MinBinSize     int
	MaxBinSize     int
	RetryInterval  int
	RetryExponent  int
	MaxRetries     int
}

// NewKadParams() returns a params struct with default values
func NewKadParams() *KadParams {
	return &KadParams{
		MaxProxDisplay: 8,
		MinProxBinSize: 2,
		MinBinSize:     2,
		MaxBinSize:     4,
		RetryInterval:  42000000000,
		MaxRetries:     42,
		RetryExponent:  2,
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
	Peer    Peer
	seenAt  time.Time
	retries int
}

func (self *KadPeer) PO(val pot.PotVal, pos int) (po int, eq bool) {
	kp, ok := val.(*KadPeer)
	var ha *pot.HashAddress
	if ok {
		ha = kp.HashAddress
	} else {
		ha = val.(*pot.HashAddress)
	}
	return self.HashAddress.PO(pot.PotVal(ha), pos)
}

func (self *KadPeer) String() string {
	if self == nil {
		return "<nil>"
	}
	// return string(self.OverlayAddr())
	return self.HashAddress.Address.String()
}

func (self *Kademlia) callable(val pot.PotVal) *KadPeer {
	kp := val.(*KadPeer)
	// not callable if peer is live or exceeded maxRetries
	if kp.Peer != nil || kp.retries > self.MaxRetries {
		glog.V(logger.Detail).Infof("peer %v not callable", kp.PeerAddr)
		return nil
	}
	// calculate the allowed number of retries based on time lapsed since last seen
	timeAgo := time.Since(kp.seenAt)
	var retries int
	for delta := int(timeAgo) / self.RetryInterval; delta > 0; delta /= self.RetryExponent {
		glog.V(logger.Detail).Infof("delta: %v", delta)
		retries++
	}
	// this is never called concurrently, so safe to increment
	// peer can be retried again
	if retries < kp.retries {
		glog.V(logger.Detail).Infof("log time needed before retry %v, wait only warrants %v", kp.retries, retries)
		return nil
	}
	kp.retries++
	glog.V(logger.Detail).Infof("peer %v is callable", kp.PeerAddr)

	return kp
}

// NewKadPeer(na) creates a kademlia peer from a PeerAddr interface
func NewKadPeer(na PeerAddr) *KadPeer {
	// o := na.OverlayAddr()
	// glog.V(logger.Detail).Infof("newkadpeer from peerAddr overlay address: %x", o[:6])
	return &KadPeer{
		HashAddress: pot.NewHashAddressFromBytes(na.OverlayAddr()),
		PeerAddr:    na,
		seenAt:      time.Now(),
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
	glog.V(logger.Detail).Infof("add peers: %v out of %v new; root %v", np.Size()-common, np.Size(), np.Pin().String()[:6])
	return nil
}

// On(p) inserts the peer as a kademlia peer into the live peers
func (self *Kademlia) On(p Peer) {
	pp := NewKadPeer(p)
	// var pp *KadPeer
	kp := pp
	self.conns.Swap(pp, func(v pot.PotVal) pot.PotVal {
		if v == nil {
			self.peers.Swap(pp, func(v pot.PotVal) pot.PotVal {
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
	kp.seenAt = time.Now()
	kp.retries = 0
	prox := self.proxLimit()

	vp, ok := kp.Peer.(KadDiscovery)
	if !ok {
		glog.V(logger.Detail).Infof("not discovery peer")
		return
	}
	// vp.NotifyProx(uint8(prox))
	f := func(val pot.PotVal, po int) {
		glog.V(logger.Detail).Infof("peer %v nofified", vp)
		dp := val.(KadDiscovery)
		dp.NotifyPeer(kp.Peer, uint8(po))
		dp.NotifyProx(uint8(prox))
	}
	self.conns.EachNeighbourAsync(pp, 255, 255, f, false)
}

// Off removes a peer from among live peers
func (self *Kademlia) Off(p Peer) {
	kp := NewKadPeer(p)
	self.conns.Swap(kp, func(v pot.PotVal) pot.PotVal {
		if v != nil {
			kp = v.(*KadPeer)
			kp.Peer = nil
			return nil
		}
		return nil
	})
	kp.Peer = nil
	kp.retries = 0
	kp.seenAt = time.Now()
}

// EachLivePeer(base, po, f) is an iterator applying f to each live peer
// that has proximity order po or less as measured from the base
// if base is nil, kademlia base address is used
func (self *Kademlia) EachLivePeer(base []byte, o int, f func(Peer, int) bool) {
	var p pot.PotVal
	if base == nil {
		p = pot.PotVal(self.addr)
	} else {
		p = pot.NewHashAddressFromBytes(base)
	}
	self.conns.EachNeighbour(p, func(val pot.PotVal, po int) bool {
		if po > o {
			return true
		}
		return f(val.(*KadPeer).Peer, po)
	})
}

// EachPeer(base, po, f) is an iterator applying f to each known peer
// that has proximity order po or less as measured from the base
// if base is nil, kademlia base address is used
func (self *Kademlia) EachPeer(base []byte, o int, f func(PeerAddr, int) bool) {
	var p pot.PotVal
	if base == nil {
		p = pot.PotVal(self.addr)
	} else {
		p = pot.NewHashAddressFromBytes(base)
	}
	self.peers.EachNeighbour(p, func(val pot.PotVal, po int) bool {
		if po > o {
			return true
		}
		return f(val.(*KadPeer).Peer, po)
	})
}

// proxLimit() returns the proximity order that defines the distance of
// the nearest neighbour set with cardinality >= MinProxBinSize
// if there is altogether less than MinProxBinSize peers it returns 0
func (self *Kademlia) proxLimit() int {
	if self.conns.Size() < self.MinProxBinSize {
		return 0
	}
	var proxLimit int
	var size int
	f := func(v pot.PotVal, i int) bool {
		size++
		proxLimit = i
		return size < self.MinProxBinSize
	}
	self.conns.EachNeighbour(pot.PotVal(self.addr), f)
	return proxLimit
}

// SuggestPeer returns a known peer for the lowest proximity bin for the
// lowest bincount below proxLimit
// naturally if there is an empty row it returns a peer for that
//
func (self *Kademlia) SuggestPeer() (p PeerAddr, o int, want bool) {
	minsize := self.MinBinSize
	proxLimit := self.proxLimit()
	// if there is a callable neighbour within the current proxBin, connect
	// this makes sure nearest neighbour set is fully connected
	glog.V(logger.Detail).Infof("candidate prox peer checking above PO %v", proxLimit)
	var ppo int
	self.peers.EachNeighbour(self.addr, func(val pot.PotVal, po int) bool {
		r := self.callable(val)
		if r == nil {
			glog.V(logger.Detail).Infof("candidate peer not callable: %#v", r)
			return po > proxLimit
		}
		p = r
		return false
	})
	if p != nil {
		glog.V(logger.Detail).Infof("candidate prox peer found: %v (%v), %#v", p, ppo, p)
		return p, 0, false
	}
	glog.V(logger.Detail).Infof("no candidate prox peers to connect to (ProxLimit: %v, minProxSize: %v)", proxLimit, self.MinProxBinSize)

	var bpo []int
	prev := -1
	self.conns.EachBin(self.addr, 0, func(po, size int, f func(func(val pot.PotVal, i int) bool) bool) bool {
		glog.V(logger.Detail).Infof("check PO%02d: ", po)
		prev++
		if po > prev {
			size = 0
			po = prev
		}
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
	// dont ask for new peers (want = false)
	// try to select a candidate peer
	for i := len(bpo) - 1; i >= 0; i-- {
		// find the first callable peer
		self.peers.EachBin(self.addr, bpo[i], func(po, size int, f func(func(val pot.PotVal, i int) bool) bool) bool {
			// for each bin we find callable candidate peers
			f(func(val pot.PotVal, i int) bool {
				r := self.callable(val)
				glog.V(logger.Detail).Infof("check PO%02d: ", po)
				if r == nil {
					return i < proxLimit
				}
				p = r
				return false
			})
			return false
		})
		// found a candidate
		if p != nil {
			break
		}
		// cannot find a candidate, ask for more for this proximity bin specifically
		o = bpo[i]
		want = true
	}
	return p, o, want
}

// kademlia table + kaddb table displayed with ascii
func (self *Kademlia) String() string {

	var rows []string

	rows = append(rows, "=========================================================================")
	rows = append(rows, fmt.Sprintf("%v KΛÐΞMLIΛ hive: queen's address: %v", time.Now().UTC().Format(time.UnixDate), self.addr.Address.String()[:6]))
	rows = append(rows, fmt.Sprintf("population: %d (%d), ProxBinSize: %d, MinBinSize: %d, MaxBinSize: %d", self.conns.Size(), self.peers.Size(), self.MinProxBinSize, self.MinBinSize, self.MaxBinSize))

	liverows := make([]string, self.MaxProxDisplay)
	peersrows := make([]string, self.MaxProxDisplay)
	var proxLimit int
	prev := -1
	var proxLimitSet bool
	rest := self.conns.Size()
	self.conns.EachBin(self.addr, 0, func(po, size int, f func(func(val pot.PotVal, i int) bool) bool) bool {
		var rowlen int
		row := []string{fmt.Sprintf("%2d", size)}
		rest -= size
		f(func(val pot.PotVal, vpo int) bool {
			row = append(row, val.(*KadPeer).String()[:6])
			rowlen++
			return rowlen < 4
		})
		if !proxLimitSet && (po > prev+1 || rest < self.MinProxBinSize) {
			proxLimitSet = true
			proxLimit = prev + 1
		}
		for ; rowlen <= 5; rowlen++ {
			row = append(row, "      ")
		}
		liverows[po] = strings.Join(row, " ")
		prev = po
		return true
	})

	self.peers.EachBin(self.addr, 0, func(po, size int, f func(func(val pot.PotVal, i int) bool) bool) bool {
		var rowlen int
		row := []string{fmt.Sprintf("%2d", size)}
		f(func(val pot.PotVal, vpo int) bool {
			kp := val.(*KadPeer)
			// if kp.Peer != nil {
			// 	return true
			// }
			row = append(row, kp.String()[:6])
			rowlen++
			return rowlen < 4
		})
		peersrows[po] = strings.Join(row, " ")
		return true
	})

	for i := 0; i < self.MaxProxDisplay; i++ {
		if i == proxLimit {
			rows = append(rows, fmt.Sprintf("============ PROX LIMIT: %d ==========================================", i))
		}
		left := liverows[i]
		right := peersrows[i]
		if len(left) == 0 {
			left = " 0                                          "
		}
		if len(right) == 0 {
			right = " 0"
		}
		rows = append(rows, fmt.Sprintf("%03d %v | %v", i, left, right))
	}
	rows = append(rows, "=========================================================================")
	return "\n" + strings.Join(rows, "\n")
}
