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

package discv5

import (
	"crypto/ecdsa"
	"fmt"
	"math/rand"

	"net"
	"reflect"
	"testing"
	"testing/quick"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type nullTransport struct{}

func (nullTransport) sendPing(remote *Node, remoteAddr *net.UDPAddr) []byte { return []byte{1} }
func (nullTransport) sendPong(remote *Node, pingHash []byte)                {}
func (nullTransport) sendFindnode(remote *Node, target NodeID)              {}
func (nullTransport) sendNeighbours(remote *Node, nodes []*Node)            {}
func (nullTransport) localAddr() *net.UDPAddr                               { return new(net.UDPAddr) }
func (nullTransport) Close()                                                {}

// func TestTable_pingReplace(t *testing.T) {
// 	doit := func(newNodeIsResponding, lastInBucketIsResponding bool) {
// 		transport := newPingRecorder()
// 		tab, _ := newTable(transport, NodeID{}, &net.UDPAddr{})
// 		defer tab.Close()
// 		pingSender := NewNode(MustHexID("a502af0f59b2aab7746995408c79e9ca312d2793cc997e44fc55eda62f0150bbb8c59a6f9269ba3a081518b62699ee807c7c19c20125ddfccca872608af9e370"), net.IP{}, 99, 99)
//
// 		// fill up the sender's bucket.
// 		last := fillBucket(tab, 253)
//
// 		// this call to bond should replace the last node
// 		// in its bucket if the node is not responding.
// 		transport.responding[last.ID] = lastInBucketIsResponding
// 		transport.responding[pingSender.ID] = newNodeIsResponding
// 		tab.bond(true, pingSender.ID, &net.UDPAddr{}, 0)
//
// 		// first ping goes to sender (bonding pingback)
// 		if !transport.pinged[pingSender.ID] {
// 			t.Error("table did not ping back sender")
// 		}
// 		if newNodeIsResponding {
// 			// second ping goes to oldest node in bucket
// 			// to see whether it is still alive.
// 			if !transport.pinged[last.ID] {
// 				t.Error("table did not ping last node in bucket")
// 			}
// 		}
//
// 		tab.mutex.Lock()
// 		defer tab.mutex.Unlock()
// 		if l := len(tab.buckets[253].entries); l != bucketSize {
// 			t.Errorf("wrong bucket size after bond: got %d, want %d", l, bucketSize)
// 		}
//
// 		if lastInBucketIsResponding || !newNodeIsResponding {
// 			if !contains(tab.buckets[253].entries, last.ID) {
// 				t.Error("last entry was removed")
// 			}
// 			if contains(tab.buckets[253].entries, pingSender.ID) {
// 				t.Error("new entry was added")
// 			}
// 		} else {
// 			if contains(tab.buckets[253].entries, last.ID) {
// 				t.Error("last entry was not removed")
// 			}
// 			if !contains(tab.buckets[253].entries, pingSender.ID) {
// 				t.Error("new entry was not added")
// 			}
// 		}
// 	}
//
// 	doit(true, true)
// 	doit(false, true)
// 	doit(true, false)
// 	doit(false, false)
// }

func TestBucket_bumpNoDuplicates(t *testing.T) {
	t.Parallel()
	cfg := &quick.Config{
		MaxCount: 1000,
		Rand:     rand.New(rand.NewSource(time.Now().Unix())),
		Values: func(args []reflect.Value, rand *rand.Rand) {
			// generate a random list of nodes. this will be the content of the bucket.
			n := rand.Intn(bucketSize-1) + 1
			nodes := make([]*Node, n)
			for i := range nodes {
				nodes[i] = nodeAtDistance(common.Hash{}, 200)
			}
			args[0] = reflect.ValueOf(nodes)
			// generate random bump positions.
			bumps := make([]int, rand.Intn(100))
			for i := range bumps {
				bumps[i] = rand.Intn(len(nodes))
			}
			args[1] = reflect.ValueOf(bumps)
		},
	}

	prop := func(nodes []*Node, bumps []int) (ok bool) {
		b := &bucket{entries: make([]*Node, len(nodes))}
		copy(b.entries, nodes)
		for i, pos := range bumps {
			b.bump(b.entries[pos])
			if hasDuplicates(b.entries) {
				t.Logf("bucket has duplicates after %d/%d bumps:", i+1, len(bumps))
				for _, n := range b.entries {
					t.Logf("  %p", n)
				}
				return false
			}
		}
		return true
	}
	if err := quick.Check(prop, cfg); err != nil {
		t.Error(err)
	}
}

// fillBucket inserts nodes into the given bucket until
// it is full. The node's IDs dont correspond to their
// hashes.
func fillBucket(tab *Table, ld int) (last *Node) {
	b := tab.buckets[ld]
	for len(b.entries) < bucketSize {
		b.entries = append(b.entries, nodeAtDistance(tab.self.sha, ld))
	}
	return b.entries[bucketSize-1]
}

// nodeAtDistance creates a node for which logdist(base, n.sha) == ld.
// The node's ID does not correspond to n.sha.
func nodeAtDistance(base common.Hash, ld int) (n *Node) {
	n = new(Node)
	n.sha = hashAtDistance(base, ld)
	copy(n.ID[:], n.sha[:]) // ensure the node still has a unique ID
	return n
}

type pingRecorder struct{ responding, pinged map[NodeID]bool }

func newPingRecorder() *pingRecorder {
	return &pingRecorder{make(map[NodeID]bool), make(map[NodeID]bool)}
}

func (t *pingRecorder) findnode(toid NodeID, toaddr *net.UDPAddr, target NodeID) ([]*Node, error) {
	panic("findnode called on pingRecorder")
}
func (t *pingRecorder) close() {}
func (t *pingRecorder) waitping(from NodeID) error {
	return nil // remote always pings
}
func (t *pingRecorder) ping(toid NodeID, toaddr *net.UDPAddr) error {
	t.pinged[toid] = true
	if t.responding[toid] {
		return nil
	} else {
		return errTimeout
	}
}

func TestTable_closest(t *testing.T) {
	t.Parallel()

	test := func(test *closeTest) bool {
		// for any node table, Target and N
		tab := newTable(test.Self, &net.UDPAddr{})
		tab.stuff(test.All)

		// check that doClosest(Target, N) returns nodes
		result := tab.closest(test.Target, test.N).entries
		if hasDuplicates(result) {
			t.Errorf("result contains duplicates")
			return false
		}
		if !sortedByDistanceTo(test.Target, result) {
			t.Errorf("result is not sorted by distance to target")
			return false
		}

		// check that the number of results is min(N, tablen)
		wantN := test.N
		if tab.count < test.N {
			wantN = tab.count
		}
		if len(result) != wantN {
			t.Errorf("wrong number of nodes: got %d, want %d", len(result), wantN)
			return false
		} else if len(result) == 0 {
			return true // no need to check distance
		}

		// check that the result nodes have minimum distance to target.
		for _, b := range tab.buckets {
			for _, n := range b.entries {
				if contains(result, n.ID) {
					continue // don't run the check below for nodes in result
				}
				farthestResult := result[len(result)-1].sha
				if distcmp(test.Target, n.sha, farthestResult) < 0 {
					t.Errorf("table contains node that is closer to target but it's not in result")
					t.Logf("  Target:          %v", test.Target)
					t.Logf("  Farthest Result: %v", farthestResult)
					t.Logf("  ID:              %v", n.ID)
					return false
				}
			}
		}
		return true
	}
	if err := quick.Check(test, quickcfg()); err != nil {
		t.Error(err)
	}
}

func TestTable_ReadRandomNodesGetAll(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 200,
		Rand:     rand.New(rand.NewSource(time.Now().Unix())),
		Values: func(args []reflect.Value, rand *rand.Rand) {
			args[0] = reflect.ValueOf(make([]*Node, rand.Intn(1000)))
		},
	}
	test := func(buf []*Node) bool {
		tab := newTable(NodeID{}, &net.UDPAddr{})
		for i := 0; i < len(buf); i++ {
			ld := cfg.Rand.Intn(len(tab.buckets))
			tab.stuff([]*Node{nodeAtDistance(tab.self.sha, ld)})
		}
		gotN := tab.readRandomNodes(buf)
		if gotN != tab.count {
			t.Errorf("wrong number of nodes, got %d, want %d", gotN, tab.count)
			return false
		}
		if hasDuplicates(buf[:gotN]) {
			t.Errorf("result contains duplicates")
			return false
		}
		return true
	}
	if err := quick.Check(test, cfg); err != nil {
		t.Error(err)
	}
}

type closeTest struct {
	Self   NodeID
	Target common.Hash
	All    []*Node
	N      int
}

func (*closeTest) Generate(rand *rand.Rand, size int) reflect.Value {
	t := &closeTest{
		Self:   gen(NodeID{}, rand).(NodeID),
		Target: gen(common.Hash{}, rand).(common.Hash),
		N:      rand.Intn(bucketSize),
	}
	for _, id := range gen([]NodeID{}, rand).([]NodeID) {
		t.All = append(t.All, &Node{ID: id})
	}
	return reflect.ValueOf(t)
}

func hasDuplicates(slice []*Node) bool {
	seen := make(map[NodeID]bool)
	for i, e := range slice {
		if e == nil {
			panic(fmt.Sprintf("nil *Node at %d", i))
		}
		if seen[e.ID] {
			return true
		}
		seen[e.ID] = true
	}
	return false
}

func sortedByDistanceTo(distbase common.Hash, slice []*Node) bool {
	var last common.Hash
	for i, e := range slice {
		if i > 0 && distcmp(distbase, e.sha, last) < 0 {
			return false
		}
		last = e.sha
	}
	return true
}

func contains(ns []*Node, id NodeID) bool {
	for _, n := range ns {
		if n.ID == id {
			return true
		}
	}
	return false
}

// gen wraps quick.Value so it's easier to use.
// it generates a random value of the given value's type.
func gen(typ interface{}, rand *rand.Rand) interface{} {
	v, ok := quick.Value(reflect.TypeOf(typ), rand)
	if !ok {
		panic(fmt.Sprintf("couldn't generate random value of type %T", typ))
	}
	return v.Interface()
}

func newkey() *ecdsa.PrivateKey {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic("couldn't generate key: " + err.Error())
	}
	return key
}
