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
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
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

If we think of random sample of items in the bins as connections in a network of
interconnected nodes than relative proximity can serve as the basis for local
decisions for graph traversal where the task is to find a route between two
points. Since in every hop, the finite distance halves, there is
a guaranteed constant maximum limit on the number of hops needed to reach one
node from the other.
*/

// KadParams holds the config params for Kademlia
type KadParams struct {
	// adjustable parameters
	MaxProxDisplay int // number of rows the table shows
	MinProxBinSize int // nearest neighbour core minimum cardinality
	MinBinSize     int // minimum number of peers in a row
	MaxBinSize     int // maximum number of peers in a row before pruning
	RetryInterval  int // initial interval before a peer is first redialed
	RetryExponent  int // exponent to multiply retry intervals with
	MaxRetries     int // maximum number of redial attempts
	PruneInterval  int // interval between peer pruning cycles
}

// NewKadParams returns a params struct with default values
func NewKadParams() *KadParams {
	return &KadParams{
		MaxProxDisplay: 8,
		MinProxBinSize: 2,
		MinBinSize:     2,
		MaxBinSize:     4,
		//RetryInterval:  42000000000,
		RetryInterval: 420000000,
		MaxRetries:    42,
		RetryExponent: 2,
	}
}

// Kademlia is a table of live peers and a db of known peers
type Kademlia struct {
	*KadParams          // Kademlia configuration parameters
	base       []byte   // immutable baseaddress of the table
	addrs      *pot.Pot // pots container for known peer addresses
	conns      *pot.Pot // pots container for live peer connections
	depth      uint8    // stores the last calculated depth
}

// NewKademlia creates a Kademlia table for base address addr
// with parameters as in params
// if params is nil, it uses default values
func NewKademlia(addr []byte, params *KadParams) *Kademlia {
	if params == nil {
		params = NewKadParams()
	}
	return &Kademlia{
		base:      addr,
		KadParams: params,
		addrs:     pot.NewPot(nil, 0),
		conns:     pot.NewPot(nil, 0),
	}
}

type Notifier interface {
	NotifyPeer(OverlayAddr, uint8) error
	NotifyDepth(uint8) error
}

// OverlayPeer interface captures the common aspect of view of a peer from the Overlay
// topology driver
type OverlayPeer interface {
	Address() []byte
}

// OverlayConn represents a connected peer
type OverlayConn interface {
	OverlayPeer
	Drop(error)       // call to indicate a peer should be expunged
	Off() OverlayAddr // call to return a persitent OverlayAddr
}

type OverlayAddr interface {
	OverlayPeer
	Update(OverlayAddr) OverlayAddr // returns the updated version of the original
}

// entry represents a Kademlia table entry (an extension of OverlayPeer)
// implements the pot.PotVal interface via BytesAddress, so entry can be
// used directly as a pot element
type entry struct {
	pot.PotVal
	OverlayPeer
	seenAt  time.Time
	retries int
}

// newEntry creates a kademlia peer from an OverlayPeer interface
func newEntry(p OverlayPeer) *entry {
	return &entry{
		PotVal:      pot.NewBytesVal(p, nil),
		OverlayPeer: p,
		seenAt:      time.Now(),
	}
}

func (self *entry) addr() OverlayAddr {
	a, _ := self.OverlayPeer.(OverlayAddr)
	return a
}

func (self *entry) conn() OverlayConn {
	c, _ := self.OverlayPeer.(OverlayConn)
	return c
}

func (self *entry) String() string {
	return fmt.Sprintf("%x", self.OverlayPeer.Address())
}

// Register enters each OverlayAddr as kademlia peer record into the
// database of known peer addresses
func (self *Kademlia) Register(peers chan OverlayAddr) error {
	np := pot.NewPot(nil, 0)
	for p := range peers {
		// error if self received, peer should know better
		if bytes.Equal(p.Address(), self.base) {
			return fmt.Errorf("add peers: %x is self", self.base)
		}
		np, _, _ = pot.Add(np, pot.PotVal(newEntry(p)))
	}
	com := self.addrs.Merge(np)
	log.Trace(fmt.Sprintf("merged %v peers, %v known", np.Size(), com))

	// TODO: remove this check
	m := make(map[string]bool)
	self.addrs.Each(func(val pot.PotVal, i int) bool {
		_, found := m[val.String()]
		if found {
			panic("duplicate found")
		}
		m[val.String()] = true
		return true
	})
	return nil
}

// SuggestPeer returns a known peer for the lowest proximity bin for the
// lowest bincount below depth
// naturally if there is an empty row it returns a peer for that
//
func (self *Kademlia) SuggestPeer() (a OverlayAddr, o int, want bool) {
	minsize := self.MinBinSize
	depth := self.Depth()
	// if there is a callable neighbour within the current proxBin, connect
	// this makes sure nearest neighbour set is fully connected
	log.Trace(fmt.Sprintf("candidate prox peer checking above PO %v", depth))
	var ppo int
	ba := pot.NewBytesVal(self.base, nil)
	self.addrs.EachNeighbour(ba, func(val pot.PotVal, po int) bool {
		a = self.callable(val)
		log.Trace(fmt.Sprintf("candidate prox peer at %x: %v (%v). a == nil is %v", val.(*entry).Address(), a, po, a == nil))
		ppo = po
		return a == nil && po >= depth
	})
	if a != nil {
		log.Trace(fmt.Sprintf("candidate prox peer found: %v (%v)", a, ppo))
		return a, 0, false
	}
	log.Trace(fmt.Sprintf("no candidate prox peers to connect to (Depth: %v, minProxSize: %v) %#v", depth, self.MinProxBinSize, a))

	var bpo []int
	prev := -1
	self.conns.EachBin(pot.NewBytesVal(self.base, nil), 0, func(po, size int, f func(func(val pot.PotVal, i int) bool) bool) bool {
		log.Trace(fmt.Sprintf("check PO%02d: ", po))
		prev++
		if po > prev {
			size = 0
			po = prev
		}
		if size < minsize {
			minsize = size
			bpo = append(bpo, po)
		}
		return size > 0 && po < depth
	})
	// all buckets are full
	// minsize == self.MinBinSize
	if len(bpo) == 0 {
		return nil, 0, false
	}
	// as long as we got candidate peers to connect to
	// dont ask for new peers (want = false)
	// try to select a candidate peer
	for i := len(bpo) - 1; i >= 0; i-- {
		// find the first callable peer
		self.addrs.EachBin(ba, bpo[i], func(po, size int, f func(func(pot.PotVal, int) bool) bool) bool {
			// for each bin we find callable candidate peers
			log.Trace(fmt.Sprintf("check PO%02d: ", po))
			f(func(val pot.PotVal, j int) bool {
				a = self.callable(val)
				return a == nil && po < depth
			})
			return false
		})
		// found a candidate
		if a != nil {
			break
		}
		// cannot find a candidate, ask for more for this proximity bin specifically
		o = bpo[i]
		want = true
	}
	return a, o, want
}

// On inserts the peer as a kademlia peer into the live peers
func (self *Kademlia) On(p OverlayConn) {
	e := newEntry(p)
	self.conns.Swap(p, func(v pot.PotVal) pot.PotVal {
		// if not found live
		if v == nil {
			// insert new online peer into addrs
			self.addrs.Swap(p, func(v pot.PotVal) pot.PotVal {
				return e
			})
			// insert new online peer into conns
			return e
		}
		// found among live peers, do nothing
		return v
	})

	log.Trace(fmt.Sprintf("Notifier:%#v", p))
	np, ok := p.(Notifier)
	if !ok {
		return
	}
	log.Trace(fmt.Sprintf("notify:%v", p))

	depth := uint8(self.Depth())
	if depth != self.depth {
		self.depth = depth
	} else {
		depth = 0
	}

	go np.NotifyDepth(depth)
	f := func(val pot.PotVal, po int) {
		dp := val.(*entry).OverlayPeer.(Notifier)
		dp.NotifyPeer(p.Off(), uint8(po))
		log.Trace(fmt.Sprintf("peer %v notified of %v (%v)", dp, p, po))
		if depth > 0 {
			dp.NotifyDepth(depth)
			log.Trace("peer %v notified of new depth %v", dp, depth)
		}
	}
	self.conns.EachNeighbourAsync(e, 1024, 255, f, false)
}

// Off removes a peer from among live peers
func (self *Kademlia) Off(p OverlayConn) {
	self.addrs.Swap(p, func(v pot.PotVal) pot.PotVal {
		// v cannot be nil, must check otherwise we overwrite entry
		if v == nil {
			panic(fmt.Sprintf("connected peer not found %v", p))
		}
		self.conns.Swap(p, func(v pot.PotVal) pot.PotVal {
			// v cannot be nil, but no need to check
			return nil
		})
		return newEntry(p.Off())
	})
}

// EachConn is an iterator with args (base, po, f) applies f to each live peer
// that has proximity order po or less as measured from the base
// if base is nil, kademlia base address is used
func (self *Kademlia) EachConn(base []byte, o int, f func(OverlayConn, int, bool) bool) {
	if len(base) == 0 {
		base = self.base
	}
	p := pot.NewBytesVal(base, nil)
	self.conns.EachNeighbour(p, func(val pot.PotVal, po int) bool {
		if po > o {
			return true
		}
		isproxbin := false
		if l, _ := p.PO(val, 0); l >= self.Depth() {
			isproxbin = true
		}
		return f(val.(*entry).conn(), po, isproxbin)
	})
}

// EachAddr(base, po, f) is an iterator applying f to each known peer
// that has proximity order po or less as measured from the base
// if base is nil, kademlia base address is used
func (self *Kademlia) EachAddr(base []byte, o int, f func(OverlayAddr, int) bool) {
	if len(base) == 0 {
		base = self.base
	}
	p := pot.NewBytesVal(base, nil)
	self.addrs.EachNeighbour(p, func(val pot.PotVal, po int) bool {
		if po > o {
			return true
		}
		return f(val.(*entry).addr(), po)
	})
}

// Depth returns the proximity order that defines the distance of
// the nearest neighbour set with cardinality >= MinProxBinSize
// if there is altogether less than MinProxBinSize peers it returns 0
func (self *Kademlia) Depth() (depth int) {
	if self.conns.Size() < self.MinProxBinSize {
		return 0
	}
	var size int
	f := func(v pot.PotVal, i int) bool {
		size++
		depth = i
		return size < self.MinProxBinSize
	}
	self.conns.EachNeighbour(pot.NewBytesVal(self.base, nil), f)
	return depth
}

func (self *Kademlia) callable(val pot.PotVal) OverlayAddr {
	e := val.(*entry)
	// not callable if peer is live or exceeded maxRetries
	if e.conn() != nil || e.retries > self.MaxRetries {
		log.Trace(fmt.Sprintf("peer %v (%T) not callable", e, e.OverlayPeer))
		return nil
	}
	// calculate the allowed number of retries based on time lapsed since last seen
	timeAgo := time.Since(e.seenAt)
	var retries int
	for delta := int(timeAgo) / self.RetryInterval; delta > 0; delta /= self.RetryExponent {
		log.Trace(fmt.Sprintf("delta: %v", delta))
		retries++
	}

	// this is never called concurrently, so safe to increment
	// peer can be retried again
	if retries < e.retries {
		log.Trace(fmt.Sprintf("long time since last try (at %v) needed before retry %v, wait only warrants %v", timeAgo, e.retries, retries))
		return nil
	}
	e.retries++
	log.Trace(fmt.Sprintf("peer %v is callable", e))

	return e.addr()
}

// BaseAddr return the kademlia base addres
func (self *Kademlia) BaseAddr() []byte {
	return self.base
}

// String returns kademlia table + kaddb table displayed with ascii
func (self *Kademlia) String() string {

	var rows []string

	rows = append(rows, "=========================================================================")
	rows = append(rows, fmt.Sprintf("%v KΛÐΞMLIΛ hive: queen's address: %x", time.Now().UTC().Format(time.UnixDate), self.BaseAddr()[:3]))
	rows = append(rows, fmt.Sprintf("population: %d (%d), MinProxBinSize: %d, MinBinSize: %d, MaxBinSize: %d", self.conns.Size(), self.addrs.Size(), self.MinProxBinSize, self.MinBinSize, self.MaxBinSize))

	liverows := make([]string, self.MaxProxDisplay)
	peersrows := make([]string, self.MaxProxDisplay)
	var depth int
	prev := -1
	var depthSet bool
	rest := self.conns.Size()
	self.conns.EachBin(pot.NewBytesVal(self.base, nil), 0, func(po, size int, f func(func(val pot.PotVal, i int) bool) bool) bool {
		var rowlen int
		if po >= self.MaxProxDisplay {
			po = self.MaxProxDisplay - 1
		}
		row := []string{fmt.Sprintf("%2d", size)}
		rest -= size
		f(func(val pot.PotVal, vpo int) bool {
			row = append(row, val.(*entry).String()[:6])
			rowlen++
			return rowlen < 4
		})
		if !depthSet && (po > prev+1 || rest < self.MinProxBinSize) {
			depthSet = true
			depth = prev + 1
		}
		for ; rowlen <= 5; rowlen++ {
			row = append(row, "      ")
		}
		liverows[po] = strings.Join(row, " ")
		prev = po
		return true
	})

	self.addrs.EachBin(pot.NewBytesVal(self.base, nil), 0, func(po, size int, f func(func(val pot.PotVal, i int) bool) bool) bool {
		var rowlen int
		if po >= self.MaxProxDisplay {
			po = self.MaxProxDisplay - 1
		}
		if size < 0 {
			panic("wtf")
		}
		row := []string{fmt.Sprintf("%2d", size)}
		// we are displaying live peers too
		f(func(val pot.PotVal, vpo int) bool {
			row = append(row, val.(*entry).String()[:6])
			rowlen++
			return rowlen < 4
		})
		peersrows[po] = strings.Join(row, " ")
		return true
	})

	for i := 0; i < self.MaxProxDisplay; i++ {
		if i == depth {
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

// Prune implements a forever loop reacting to a ticker time channel given
// as the first argument
// the loop quits if the channel is closed
// it checks each kademlia bin and if the peer count is higher than
// the MaxBinSize parameter it drops the oldest n peers such that
// the bin is reduced to MinBinSize peers thus leaving slots to newly
// connecting peers
func (self *Kademlia) Prune(c <-chan time.Time) {
	go func() {
		for range c {
			total := 0
			self.conns.EachBin(pot.NewBytesVal(self.base, nil), 0, func(po, size int, f func(func(pot.PotVal, int) bool) bool) bool {
				extra := size - self.MinBinSize
				if size > self.MaxBinSize {
					n := 0
					f(func(v pot.PotVal, po int) bool {
						v.(*entry).conn().Drop(fmt.Errorf("bucket full"))
						n++
						return n < extra
					})
					total += extra
				}
				return true
			})
			log.Debug(fmt.Sprintf("pruned %v peers", total))
		}
	}()
}
