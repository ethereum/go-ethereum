package discover

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"testing"
	"testing/quick"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestTable_bumpOrAddPingReplace(t *testing.T) {
	pingC := make(pingC)
	tab := newTable(pingC, NodeID{}, &net.UDPAddr{})
	last := fillBucket(tab, 200)

	// this bumpOrAdd should not replace the last node
	// because the node replies to ping.
	new := tab.bumpOrAdd(randomID(tab.self.ID, 200), &net.UDPAddr{})

	pinged := <-pingC
	if pinged != last.ID {
		t.Fatalf("pinged wrong node: %v\nwant %v", pinged, last.ID)
	}

	tab.mutex.Lock()
	defer tab.mutex.Unlock()
	if l := len(tab.buckets[200].entries); l != bucketSize {
		t.Errorf("wrong bucket size after bumpOrAdd: got %d, want %d", bucketSize, l)
	}
	if !contains(tab.buckets[200].entries, last.ID) {
		t.Error("last entry was removed")
	}
	if contains(tab.buckets[200].entries, new.ID) {
		t.Error("new entry was added")
	}
}

func TestTable_bumpOrAddPingTimeout(t *testing.T) {
	tab := newTable(pingC(nil), NodeID{}, &net.UDPAddr{})
	last := fillBucket(tab, 200)

	// this bumpOrAdd should replace the last node
	// because the node does not reply to ping.
	new := tab.bumpOrAdd(randomID(tab.self.ID, 200), &net.UDPAddr{})

	// wait for async bucket update. damn. this needs to go away.
	time.Sleep(2 * time.Millisecond)

	tab.mutex.Lock()
	defer tab.mutex.Unlock()
	if l := len(tab.buckets[200].entries); l != bucketSize {
		t.Errorf("wrong bucket size after bumpOrAdd: got %d, want %d", bucketSize, l)
	}
	if contains(tab.buckets[200].entries, last.ID) {
		t.Error("last entry was not removed")
	}
	if !contains(tab.buckets[200].entries, new.ID) {
		t.Error("new entry was not added")
	}
}

func fillBucket(tab *Table, ld int) (last *Node) {
	b := tab.buckets[ld]
	for len(b.entries) < bucketSize {
		b.entries = append(b.entries, &Node{ID: randomID(tab.self.ID, ld)})
	}
	return b.entries[bucketSize-1]
}

type pingC chan NodeID

func (t pingC) findnode(n *Node, target NodeID) ([]*Node, error) {
	panic("findnode called on pingRecorder")
}
func (t pingC) close() {
	panic("close called on pingRecorder")
}
func (t pingC) ping(n *Node) error {
	if t == nil {
		return errTimeout
	}
	t <- n.ID
	return nil
}

func TestTable_bump(t *testing.T) {
	tab := newTable(nil, NodeID{}, &net.UDPAddr{})

	// add an old entry and two recent ones
	oldactive := time.Now().Add(-2 * time.Minute)
	old := &Node{ID: randomID(tab.self.ID, 200), active: oldactive}
	others := []*Node{
		&Node{ID: randomID(tab.self.ID, 200), active: time.Now()},
		&Node{ID: randomID(tab.self.ID, 200), active: time.Now()},
	}
	tab.add(append(others, old))
	if tab.buckets[200].entries[0] == old {
		t.Fatal("old entry is at front of bucket")
	}

	// bumping the old entry should move it to the front
	tab.bump(old.ID)
	if old.active == oldactive {
		t.Error("activity timestamp not updated")
	}
	if tab.buckets[200].entries[0] != old {
		t.Errorf("bumped entry did not move to the front of bucket")
	}
}

func TestTable_closest(t *testing.T) {
	t.Parallel()

	test := func(test *closeTest) bool {
		// for any node table, Target and N
		tab := newTable(nil, test.Self, &net.UDPAddr{})
		tab.add(test.All)

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
		if tlen := tab.len(); tlen < test.N {
			wantN = tlen
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
				farthestResult := result[len(result)-1].ID
				if distcmp(test.Target, n.ID, farthestResult) < 0 {
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
	if err := quick.Check(test, quickcfg); err != nil {
		t.Error(err)
	}
}

type closeTest struct {
	Self   NodeID
	Target NodeID
	All    []*Node
	N      int
}

func (*closeTest) Generate(rand *rand.Rand, size int) reflect.Value {
	t := &closeTest{
		Self:   gen(NodeID{}, rand).(NodeID),
		Target: gen(NodeID{}, rand).(NodeID),
		N:      rand.Intn(bucketSize),
	}
	for _, id := range gen([]NodeID{}, rand).([]NodeID) {
		t.All = append(t.All, &Node{ID: id})
	}
	return reflect.ValueOf(t)
}

func TestTable_Lookup(t *testing.T) {
	self := gen(NodeID{}, quickrand).(NodeID)
	target := randomID(self, 200)
	transport := findnodeOracle{t, target}
	tab := newTable(transport, self, &net.UDPAddr{})

	// lookup on empty table returns no nodes
	if results := tab.Lookup(target); len(results) > 0 {
		t.Fatalf("lookup on empty table returned %d results: %#v", len(results), results)
	}
	// seed table with initial node (otherwise lookup will terminate immediately)
	tab.bumpOrAdd(randomID(target, 200), &net.UDPAddr{Port: 200})

	results := tab.Lookup(target)
	t.Logf("results:")
	for _, e := range results {
		t.Logf("  ld=%d, %v", logdist(target, e.ID), e.ID)
	}
	if len(results) != bucketSize {
		t.Errorf("wrong number of results: got %d, want %d", len(results), bucketSize)
	}
	if hasDuplicates(results) {
		t.Errorf("result set contains duplicate entries")
	}
	if !sortedByDistanceTo(target, results) {
		t.Errorf("result set not sorted by distance to target")
	}
	if !contains(results, target) {
		t.Errorf("result set does not contain target")
	}
}

// findnode on this transport always returns at least one node
// that is one bucket closer to the target.
type findnodeOracle struct {
	t      *testing.T
	target NodeID
}

func (t findnodeOracle) findnode(n *Node, target NodeID) ([]*Node, error) {
	t.t.Logf("findnode query at dist %d", n.DiscPort)
	// current log distance is encoded in port number
	var result []*Node
	switch n.DiscPort {
	case 0:
		panic("query to node at distance 0")
	default:
		// TODO: add more randomness to distances
		next := n.DiscPort - 1
		for i := 0; i < bucketSize; i++ {
			result = append(result, &Node{ID: randomID(t.target, next), DiscPort: next})
		}
	}
	return result, nil
}

func (t findnodeOracle) close() {}

func (t findnodeOracle) ping(n *Node) error {
	return errors.New("ping is not supported by this transport")
}

func hasDuplicates(slice []*Node) bool {
	seen := make(map[NodeID]bool)
	for _, e := range slice {
		if seen[e.ID] {
			return true
		}
		seen[e.ID] = true
	}
	return false
}

func sortedByDistanceTo(distbase NodeID, slice []*Node) bool {
	var last NodeID
	for i, e := range slice {
		if i > 0 && distcmp(distbase, e.ID, last) < 0 {
			return false
		}
		last = e.ID
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
