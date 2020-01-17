// Copyright 2018 The go-ethereum Authors
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
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"reflect"
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
	go tab.loop()
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
	records      map[enode.ID]*enode.Node
	n            *enode.Node
}

func newPingRecorder() *pingRecorder {
	var r enr.Record
	r.Set(enr.IP{0, 0, 0, 0})
	n := enode.SignNull(&r, enode.ID{})

	return &pingRecorder{
		dead:    make(map[enode.ID]bool),
		pinged:  make(map[enode.ID]bool),
		records: make(map[enode.ID]*enode.Node),
		n:       n,
	}
}

// setRecord updates a node record. Future calls to ping and
// requestENR will return this record.
func (t *pingRecorder) updateRecord(n *enode.Node) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.records[n.ID()] = n
}

// Stubs to satisfy the transport interface.
func (t *pingRecorder) Self() *enode.Node           { return nullNode }
func (t *pingRecorder) lookupSelf() []*enode.Node   { return nil }
func (t *pingRecorder) lookupRandom() []*enode.Node { return nil }
func (t *pingRecorder) close()                      {}

// ping simulates a ping request.
func (t *pingRecorder) ping(n *enode.Node) (seq uint64, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.pinged[n.ID()] = true
	if t.dead[n.ID()] {
		return 0, errTimeout
	}
	if t.records[n.ID()] != nil {
		seq = t.records[n.ID()].Seq()
	}
	return seq, nil
}

// requestENR simulates an ENR request.
func (t *pingRecorder) RequestENR(n *enode.Node) (*enode.Node, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.dead[n.ID()] || t.records[n.ID()] == nil {
		return nil, errTimeout
	}
	return t.records[n.ID()], nil
}

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

func checkNodesEqual(got, want []*enode.Node) error {
	if reflect.DeepEqual(got, want) {
		return nil
	}
	output := new(bytes.Buffer)
	fmt.Fprintf(output, "got %d nodes:\n", len(got))
	for _, n := range got {
		fmt.Fprintf(output, "  %v %v\n", n.ID(), n)
	}
	fmt.Fprintf(output, "want %d:\n", len(want))
	for _, n := range want {
		fmt.Fprintf(output, "  %v %v\n", n.ID(), n)
	}
	return errors.New(output.String())
}

func sortByID(nodes []*enode.Node) {
	sort.Slice(nodes, func(i, j int) bool {
		return string(nodes[i].ID().Bytes()) < string(nodes[j].ID().Bytes())
	})
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
