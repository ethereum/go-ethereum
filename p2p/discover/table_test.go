package discover

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"reflect"
	"testing"
	"testing/quick"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

var (
	quickrand = rand.New(rand.NewSource(time.Now().Unix()))
	quickcfg  = &quick.Config{MaxCount: 5000, Rand: quickrand}
)

func TestHexID(t *testing.T) {
	ref := NodeID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 106, 217, 182, 31, 165, 174, 1, 67, 7, 235, 220, 150, 66, 83, 173, 205, 159, 44, 10, 57, 42, 161, 26, 188}
	id1 := HexID("0x000000000000000000000000000000000000000000000000000000000000000000000000000000806ad9b61fa5ae014307ebdc964253adcd9f2c0a392aa11abc")
	id2 := HexID("000000000000000000000000000000000000000000000000000000000000000000000000000000806ad9b61fa5ae014307ebdc964253adcd9f2c0a392aa11abc")

	if id1 != ref {
		t.Errorf("wrong id1\ngot  %v\nwant %v", id1[:], ref[:])
	}
	if id2 != ref {
		t.Errorf("wrong id2\ngot  %v\nwant %v", id2[:], ref[:])
	}
}

func TestNodeID_recover(t *testing.T) {
	prv := newkey()
	hash := make([]byte, 32)
	sig, err := crypto.Sign(hash, prv)
	if err != nil {
		t.Fatalf("signing error: %v", err)
	}

	pub := newNodeID(prv)
	recpub, err := recoverNodeID(hash, sig)
	if err != nil {
		t.Fatalf("recovery error: %v", err)
	}
	if pub != recpub {
		t.Errorf("recovered wrong pubkey:\ngot:  %v\nwant: %v", recpub, pub)
	}
}

func TestNodeID_distcmp(t *testing.T) {
	distcmpBig := func(target, a, b NodeID) int {
		tbig := new(big.Int).SetBytes(target[:])
		abig := new(big.Int).SetBytes(a[:])
		bbig := new(big.Int).SetBytes(b[:])
		return new(big.Int).Xor(tbig, abig).Cmp(new(big.Int).Xor(tbig, bbig))
	}
	if err := quick.CheckEqual(distcmp, distcmpBig, quickcfg); err != nil {
		t.Error(err)
	}
}

// the random tests is likely to miss the case where they're equal.
func TestNodeID_distcmpEqual(t *testing.T) {
	base := NodeID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	x := NodeID{15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	if distcmp(base, x, x) != 0 {
		t.Errorf("distcmp(base, x, x) != 0")
	}
}

func TestNodeID_logdist(t *testing.T) {
	logdistBig := func(a, b NodeID) int {
		abig, bbig := new(big.Int).SetBytes(a[:]), new(big.Int).SetBytes(b[:])
		return new(big.Int).Xor(abig, bbig).BitLen()
	}
	if err := quick.CheckEqual(logdist, logdistBig, quickcfg); err != nil {
		t.Error(err)
	}
}

// the random tests is likely to miss the case where they're equal.
func TestNodeID_logdistEqual(t *testing.T) {
	x := NodeID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	if logdist(x, x) != 0 {
		t.Errorf("logdist(x, x) != 0")
	}
}

func TestNodeID_randomID(t *testing.T) {
	// we don't use quick.Check here because its output isn't
	// very helpful when the test fails.
	for i := 0; i < quickcfg.MaxCount; i++ {
		a := gen(NodeID{}, quickrand).(NodeID)
		dist := quickrand.Intn(len(NodeID{}) * 8)
		result := randomID(a, dist)
		actualdist := logdist(result, a)

		if dist != actualdist {
			t.Log("a:     ", a)
			t.Log("result:", result)
			t.Fatalf("#%d: distance of result is %d, want %d", i, actualdist, dist)
		}
	}
}

func (NodeID) Generate(rand *rand.Rand, size int) reflect.Value {
	var id NodeID
	m := rand.Intn(len(id))
	for i := len(id) - 1; i > m; i-- {
		id[i] = byte(rand.Uint32())
	}
	return reflect.ValueOf(id)
}

func TestTable_bumpOrAddPingReplace(t *testing.T) {
	pingC := make(pingC)
	tab := newTable(pingC, NodeID{}, &net.UDPAddr{})
	last := fillBucket(tab, 200)

	// this bumpOrAdd should not replace the last node
	// because the node replies to ping.
	new := tab.bumpOrAdd(randomID(tab.self.ID, 200), nil)

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
	new := tab.bumpOrAdd(randomID(tab.self.ID, 200), nil)

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
	t.t.Logf("findnode query at dist %d", n.Addr.Port)
	// current log distance is encoded in port number
	var result []*Node
	switch port := n.Addr.Port; port {
	case 0:
		panic("query to node at distance 0")
	case 1:
		result = append(result, &Node{ID: t.target, Addr: &net.UDPAddr{Port: 0}})
	default:
		// TODO: add more randomness to distances
		port--
		for i := 0; i < bucketSize; i++ {
			result = append(result, &Node{ID: randomID(t.target, port), Addr: &net.UDPAddr{Port: port}})
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
