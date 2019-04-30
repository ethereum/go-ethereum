// Copyright 2015 The go-ethereum Authors
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

package discover

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
)

var nullNode *enode.Node

func init() {
	var r enr.Record
	r.Set(enr.IP{0, 0, 0, 0})
	nullNode = enode.SignNull(&r, enode.ID{})
}

func newTestTable(t transport) (*Table, *enode.DB) {
	db, _ := enode.OpenDB("")
	tab, _ := newTable(t, db, nil, log.Root())
	return tab, db
}

// nodeAtDistance creates a node for which enode.LogDist(base, n.id) == ld.
func nodeAtDistance(base enode.ID, ld int, ip net.IP) *node {
	var r enr.Record
	r.Set(enr.IP(ip))
	return wrapNode(enode.SignNull(&r, idAtDistance(base, ld)))
}

// idAtDistance returns a random hash such that enode.LogDist(a, b) == n
func idAtDistance(a enode.ID, n int) (b enode.ID) {
	if n == 0 {
		return a
	}
	// flip bit at position n, fill the rest with random bits
	b = a
	pos := len(a) - n/8 - 1
	bit := byte(0x01) << (byte(n%8) - 1)
	if bit == 0 {
		pos++
		bit = 0x80
	}
	b[pos] = a[pos]&^bit | ^a[pos]&bit // TODO: randomize end bits
	for i := pos + 1; i < len(a); i++ {
		b[i] = byte(rand.Intn(255))
	}
	return b
}

func intIP(i int) net.IP {
	return net.IP{byte(i), 0, 2, byte(i)}
}

// fillBucket inserts nodes into the given bucket until it is full.
func fillBucket(tab *Table, n *node) (last *node) {
	ld := enode.LogDist(tab.self().ID(), n.ID())
	b := tab.bucket(n.ID())
	for len(b.entries) < bucketSize {
		b.entries = append(b.entries, nodeAtDistance(tab.self().ID(), ld, intIP(ld)))
	}
	return b.entries[bucketSize-1]
}

// fillTable adds nodes the table to the end of their corresponding bucket
// if the bucket is not full. The caller must not hold tab.mutex.
func fillTable(tab *Table, nodes []*node) {
	for _, n := range nodes {
		tab.addSeenNode(n)
	}
}

type pingRecorder struct {
	mu           sync.Mutex
	dead, pinged map[enode.ID]bool
	n            *enode.Node
}

func newPingRecorder() *pingRecorder {
	var r enr.Record
	r.Set(enr.IP{0, 0, 0, 0})
	n := enode.SignNull(&r, enode.ID{})

	return &pingRecorder{
		dead:   make(map[enode.ID]bool),
		pinged: make(map[enode.ID]bool),
		n:      n,
	}
}

func (t *pingRecorder) Self() *enode.Node {
	return nullNode
}

func (t *pingRecorder) ping(n *enode.Node) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.pinged[n.ID()] = true
	if t.dead[n.ID()] {
		return errTimeout
	} else {
		return nil
	}
}

func (t *pingRecorder) lookupSelf() []*enode.Node {
	return nil
}

func (t *pingRecorder) lookupRandom() []*enode.Node {
	return nil
}

func (t *pingRecorder) close() {}

func hasDuplicates(slice []*node) bool {
	seen := make(map[enode.ID]bool)
	for i, e := range slice {
		if e == nil {
			panic(fmt.Sprintf("nil *Node at %d", i))
		}
		if seen[e.ID()] {
			return true
		}
		seen[e.ID()] = true
	}
	return false
}

func sortedByDistanceTo(distbase enode.ID, slice []*node) bool {
	return sort.SliceIsSorted(slice, func(i, j int) bool {
		return enode.DistCmp(distbase, slice[i].ID(), slice[j].ID()) < 0
	})
}

func hexEncPrivkey(h string) *ecdsa.PrivateKey {
	b, err := hex.DecodeString(h)
	if err != nil {
		panic(err)
	}
	key, err := crypto.ToECDSA(b)
	if err != nil {
		panic(err)
	}
	return key
}

func hexEncPubkey(h string) (ret encPubkey) {
	b, err := hex.DecodeString(h)
	if err != nil {
		panic(err)
	}
	if len(b) != len(ret) {
		panic("invalid length")
	}
	copy(ret[:], b)
	return ret
}
