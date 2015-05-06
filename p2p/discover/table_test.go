package discover

import (
	"crypto/ecdsa"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestTable_pingReplace(t *testing.T) {
	doit := func(newNodeIsResponding, lastInBucketIsResponding bool) {
		transport := newPingRecorder()
		tab := newTable(transport, NodeID{}, &net.UDPAddr{}, "")
		last := fillBucket(tab, 200)
		pingSender := randomID(tab.self.ID, 200)

		// this gotPing should replace the last node
		// if the last node is not responding.
		transport.responding[last.ID] = lastInBucketIsResponding
		transport.responding[pingSender] = newNodeIsResponding
		tab.bond(true, pingSender, &net.UDPAddr{}, 0)

		// first ping goes to sender (bonding pingback)
		if !transport.pinged[pingSender] {
			t.Error("table did not ping back sender")
		}
		if newNodeIsResponding {
			// second ping goes to oldest node in bucket
			// to see whether it is still alive.
			if !transport.pinged[last.ID] {
				t.Error("table did not ping last node in bucket")
			}
		}

		tab.mutex.Lock()
		defer tab.mutex.Unlock()
		if l := len(tab.buckets[200].entries); l != bucketSize {
			t.Errorf("wrong bucket size after gotPing: got %d, want %d", bucketSize, l)
		}

		if lastInBucketIsResponding || !newNodeIsResponding {
			if !contains(tab.buckets[200].entries, last.ID) {
				t.Error("last entry was removed")
			}
			if contains(tab.buckets[200].entries, pingSender) {
				t.Error("new entry was added")
			}
		} else {
			if contains(tab.buckets[200].entries, last.ID) {
				t.Error("last entry was not removed")
			}
			if !contains(tab.buckets[200].entries, pingSender) {
				t.Error("new entry was not added")
			}
		}
	}

	doit(true, true)
	doit(false, true)
	doit(false, true)
	doit(false, false)
}

func TestBucket_bumpNoDuplicates(t *testing.T) {
	t.Parallel()
	cfg := &quick.Config{
		MaxCount: 1000,
		Rand:     quickrand,
		Values: func(args []reflect.Value, rand *rand.Rand) {
			// generate a random list of nodes. this will be the content of the bucket.
			n := rand.Intn(bucketSize-1) + 1
			nodes := make([]*Node, n)
			for i := range nodes {
				nodes[i] = &Node{ID: randomID(NodeID{}, 200)}
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

func fillBucket(tab *Table, ld int) (last *Node) {
	b := tab.buckets[ld]
	for len(b.entries) < bucketSize {
		b.entries = append(b.entries, &Node{ID: randomID(tab.self.ID, ld)})
	}
	return b.entries[bucketSize-1]
}

type pingRecorder struct{ responding, pinged map[NodeID]bool }

func newPingRecorder() *pingRecorder {
	return &pingRecorder{make(map[NodeID]bool), make(map[NodeID]bool)}
}

func (t *pingRecorder) findnode(toid NodeID, toaddr *net.UDPAddr, target NodeID) ([]*Node, error) {
	panic("findnode called on pingRecorder")
}
func (t *pingRecorder) close() {
	panic("close called on pingRecorder")
}
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
		tab := newTable(nil, test.Self, &net.UDPAddr{}, "")
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
	tab := newTable(transport, self, &net.UDPAddr{}, "")

	// lookup on empty table returns no nodes
	if results := tab.Lookup(target); len(results) > 0 {
		t.Fatalf("lookup on empty table returned %d results: %#v", len(results), results)
	}
	// seed table with initial node (otherwise lookup will terminate immediately)
	tab.add([]*Node{newNode(randomID(target, 200), &net.UDPAddr{Port: 200})})

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

func (t findnodeOracle) findnode(toid NodeID, toaddr *net.UDPAddr, target NodeID) ([]*Node, error) {
	t.t.Logf("findnode query at dist %d", toaddr.Port)
	// current log distance is encoded in port number
	var result []*Node
	switch toaddr.Port {
	case 0:
		panic("query to node at distance 0")
	default:
		// TODO: add more randomness to distances
		next := uint16(toaddr.Port) - 1
		for i := 0; i < bucketSize; i++ {
			result = append(result, &Node{ID: randomID(t.target, int(next)), UDP: next})
		}
	}
	return result, nil
}

func (t findnodeOracle) close()                                      {}
func (t findnodeOracle) waitping(from NodeID) error                  { return nil }
func (t findnodeOracle) ping(toid NodeID, toaddr *net.UDPAddr) error { return nil }

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
