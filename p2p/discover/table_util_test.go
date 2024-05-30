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
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
)

var nullNode *enode.Node

func init() {
	var r enr.Record
	r.Set(enr.IP{0, 0, 0, 0})
	nullNode = enode.SignNull(&r, enode.ID{})
}

func newTestTable(t transport, cfg Config) (*Table, *enode.DB) {
	tab, db := newInactiveTestTable(t, cfg)
	go tab.loop()
	return tab, db
}

// newInactiveTestTable creates a Table without running the main loop.
func newInactiveTestTable(t transport, cfg Config) (*Table, *enode.DB) {
	db, _ := enode.OpenDB("")
	tab, _ := newTable(t, db, cfg)
	return tab, db
}

// nodeAtDistance creates a node for which enode.LogDist(base, n.id) == ld.
func nodeAtDistance(base enode.ID, ld int, ip net.IP) *enode.Node {
	var r enr.Record
	r.Set(enr.IP(ip))
	r.Set(enr.UDP(30303))
	return enode.SignNull(&r, idAtDistance(base, ld))
}

// nodesAtDistance creates n nodes for which enode.LogDist(base, node.ID()) == ld.
func nodesAtDistance(base enode.ID, ld int, n int) []*enode.Node {
	results := make([]*enode.Node, n)
	for i := range results {
		results[i] = nodeAtDistance(base, ld, intIP(i))
	}
	return results
}

func nodesToRecords(nodes []*enode.Node) []*enr.Record {
	records := make([]*enr.Record, len(nodes))
	for i := range nodes {
		records[i] = nodes[i].Record()
	}
	return records
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

// intIP returns a LAN IP address based on i.
func intIP(i int) net.IP {
	return net.IP{10, 0, byte(i >> 8), byte(i & 0xFF)}
}

// fillBucket inserts nodes into the given bucket until it is full.
func fillBucket(tab *Table, id enode.ID) (last *tableNode) {
	ld := enode.LogDist(tab.self().ID(), id)
	b := tab.bucket(id)
	for len(b.entries) < bucketSize {
		node := nodeAtDistance(tab.self().ID(), ld, intIP(ld))
		if !tab.addFoundNode(node, false) {
			panic("node not added")
		}
	}
	return b.entries[bucketSize-1]
}

// fillTable adds nodes the table to the end of their corresponding bucket
// if the bucket is not full. The caller must not hold tab.mutex.
func fillTable(tab *Table, nodes []*enode.Node, setLive bool) {
	for _, n := range nodes {
		tab.addFoundNode(n, setLive)
	}
}

type pingRecorder struct {
	mu      sync.Mutex
	cond    *sync.Cond
	dead    map[enode.ID]bool
	records map[enode.ID]*enode.Node
	pinged  []*enode.Node
	n       *enode.Node
}

func newPingRecorder() *pingRecorder {
	var r enr.Record
	r.Set(enr.IP{0, 0, 0, 0})
	n := enode.SignNull(&r, enode.ID{})

	t := &pingRecorder{
		dead:    make(map[enode.ID]bool),
		records: make(map[enode.ID]*enode.Node),
		n:       n,
	}
	t.cond = sync.NewCond(&t.mu)
	return t
}

// updateRecord updates a node record. Future calls to ping and
// RequestENR will return this record.
func (t *pingRecorder) updateRecord(n *enode.Node) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.records[n.ID()] = n
}

// Stubs to satisfy the transport interface.
func (t *pingRecorder) Self() *enode.Node           { return nullNode }
func (t *pingRecorder) lookupSelf() []*enode.Node   { return nil }
func (t *pingRecorder) lookupRandom() []*enode.Node { return nil }

func (t *pingRecorder) waitPing(timeout time.Duration) *enode.Node {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Wake up the loop on timeout.
	var timedout atomic.Bool
	timer := time.AfterFunc(timeout, func() {
		timedout.Store(true)
		t.cond.Broadcast()
	})
	defer timer.Stop()

	// Wait for a ping.
	for {
		if timedout.Load() {
			return nil
		}
		if len(t.pinged) > 0 {
			n := t.pinged[0]
			t.pinged = append(t.pinged[:0], t.pinged[1:]...)
			return n
		}
		t.cond.Wait()
	}
}

// ping simulates a ping request.
func (t *pingRecorder) ping(n *enode.Node) (seq uint64, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.pinged = append(t.pinged, n)
	t.cond.Broadcast()

	if t.dead[n.ID()] {
		return 0, errTimeout
	}
	if t.records[n.ID()] != nil {
		seq = t.records[n.ID()].Seq()
	}
	return seq, nil
}

// RequestENR simulates an ENR request.
func (t *pingRecorder) RequestENR(n *enode.Node) (*enode.Node, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.dead[n.ID()] || t.records[n.ID()] == nil {
		return nil, errTimeout
	}
	return t.records[n.ID()], nil
}

func hasDuplicates(slice []*enode.Node) bool {
	seen := make(map[enode.ID]bool, len(slice))
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

// checkNodesEqual checks whether the two given node lists contain the same nodes.
func checkNodesEqual(got, want []*enode.Node) error {
	if len(got) == len(want) {
		for i := range got {
			if !nodeEqual(got[i], want[i]) {
				goto NotEqual
			}
		}
	}
	return nil

NotEqual:
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

func nodeEqual(n1 *enode.Node, n2 *enode.Node) bool {
	return n1.ID() == n2.ID() && n1.IPAddr() == n2.IPAddr()
}

func sortByID[N nodeType](nodes []N) {
	slices.SortFunc(nodes, func(a, b N) int {
		return bytes.Compare(a.ID().Bytes(), b.ID().Bytes())
	})
}

func sortedByDistanceTo(distbase enode.ID, slice []*enode.Node) bool {
	return slices.IsSortedFunc(slice, func(a, b *enode.Node) int {
		return enode.DistCmp(distbase, a.ID(), b.ID())
	})
}

// hexEncPrivkey decodes h as a private key.
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

// hexEncPubkey decodes h as a public key.
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

type nodeEventRecorder struct {
	evc chan recordedNodeEvent
}

type recordedNodeEvent struct {
	node  *tableNode
	added bool
}

func newNodeEventRecorder(buffer int) *nodeEventRecorder {
	return &nodeEventRecorder{
		evc: make(chan recordedNodeEvent, buffer),
	}
}

func (set *nodeEventRecorder) nodeAdded(b *bucket, n *tableNode) {
	select {
	case set.evc <- recordedNodeEvent{n, true}:
	default:
		panic("no space in event buffer")
	}
}

func (set *nodeEventRecorder) nodeRemoved(b *bucket, n *tableNode) {
	select {
	case set.evc <- recordedNodeEvent{n, false}:
	default:
		panic("no space in event buffer")
	}
}

func (set *nodeEventRecorder) waitNodePresent(id enode.ID, timeout time.Duration) bool {
	return set.waitNodeEvent(id, timeout, true)
}

func (set *nodeEventRecorder) waitNodeAbsent(id enode.ID, timeout time.Duration) bool {
	return set.waitNodeEvent(id, timeout, false)
}

func (set *nodeEventRecorder) waitNodeEvent(id enode.ID, timeout time.Duration, added bool) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case ev := <-set.evc:
			if ev.node.ID() == id && ev.added == added {
				return true
			}
		case <-timer.C:
			return false
		}
	}
}
