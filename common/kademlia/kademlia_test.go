package kademlia

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"reflect"
	"sync"
	"testing"
	"testing/quick"
	"time"

	"github.com/ethereum/go-ethereum/logger"
)

var (
	quickrand = rand.New(rand.NewSource(time.Now().Unix()))
	quickcfg  = &quick.Config{MaxCount: 5000, Rand: quickrand}
)

var once sync.Once

func LogInit(l logger.LogLevel) {
	once.Do(func() {
		logger.NewStdLogSystem(os.Stderr, log.LstdFlags, l)
	})
}

type testNode struct {
	addr Address
}

func (n *testNode) String() string {
	return fmt.Sprintf("%x", n.addr[:])
}

func (n *testNode) Addr() Address {
	return n.addr
}

func (n *testNode) Drop() {
}

func (n *testNode) Url() string {
	return ""
}

func (n *testNode) LastActive() time.Time {
	return time.Now()
}

func (n *testNode) Add(a Address) (err error) {
	return nil
}

func TestAddNode(t *testing.T) {
	LogInit(logger.DebugLevel)
	addr, ok := gen(Address{}, quickrand).(Address)
	other, ok := gen(Address{}, quickrand).(Address)
	if !ok {
		t.Errorf("oops")
	}
	kad := New(addr)
	kad.Start()
	err := kad.AddNode(&testNode{addr: other})
	_ = err
}

func TestGetNodes(t *testing.T) {
	t.Parallel()
	LogInit(logger.DebugLevel)

	test := func(test *getNodesTest) bool {
		// for any node kad.le, Target and N
		kad := New(test.Self)
		kad.MaxProx = 10
		kad.Start()
		var err error
		t.Logf("getNodesTest %v: %v\n", len(test.All), test)
		for _, node := range test.All {
			err = kad.AddNode(node)
			if err != nil {
				t.Errorf("backend not accepting node")
				return false
			}
		}

		if len(test.All) == 0 || test.N == 0 {
			return true
		}
		nodes := kad.GetNodes(test.Target, test.N)

		// check that the number of results is min(N, kad.len)
		wantN := test.N
		if tlen := kad.Count(); tlen < test.N {
			wantN = tlen
		}

		if len(nodes) != wantN {
			t.Errorf("wrong number of nodes: got %d, want %d", len(nodes), wantN)
			return false
		}

		if hasDuplicates(nodes) {
			t.Errorf("result contains duplicates")
			return false
		}

		if !sortedByDistanceTo(test.Target, nodes) {
			t.Errorf("result is not sorted by distance to target")
			return false
		}

		// check that the result nodes have minimum distance to target.
		farthestResult := nodes[len(nodes)-1].Addr()
		for i, b := range kad.buckets {
			for j, n := range b.nodes {
				if contains(nodes, n.Addr()) {
					continue // don't run the check below for nodes in result
				}
				if proxCmp(test.Target, n.Addr(), farthestResult) < 0 {
					t.Errorf("kad.le contains node that is closer to target but it's not in result")
					t.Logf("bucket %v, item %v\n", i, j)
					t.Logf("  Target:          %x", test.Target)
					t.Logf("  Farthest Result: %x", farthestResult)
					t.Logf("  ID:              %x (%d)", n.Addr(), kad.proximityBin(n.Addr()))
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

type proxTest struct {
	add     bool
	index   int
	address Address
}

var (
	addresses []Address
)

func TestProxAdjust(t *testing.T) {
	t.Parallel()
	LogInit(logger.DebugLevel)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	self := gen(Address{}, r).(Address)

	kad := New(self)
	kad.MaxProx = 10
	kad.Start()
	var err error
	for i := 0; i < 100; i++ {
		a := gen(Address{}, r).(Address)
		addresses = append(addresses, a)
		err = kad.AddNode(&testNode{addr: a})
		if err != nil {
			t.Errorf("backend not accepting node")
			return
		}
		if !kad.proxCheck(t) {
			return
		}
	}

	test := func(test *proxTest) bool {
		node := &testNode{test.address}
		if test.add {
			kad.AddNode(node)
		} else {
			kad.RemoveNode(node)
		}
		return kad.proxCheck(t)
	}
	if err := quick.Check(test, quickcfg); err != nil {
		t.Error(err)
	}
}

func TestCallback(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	self := gen(Address{}, r).(Address)
	kad := New(self)
	var bucket int
	var called chan bool
	kad.MaxProx = 5
	kad.BucketSize = 2
	kad.GetNode = func(b int) {
		bucket = b
		close(called)
	}
	kad.Start()
	var err error

	for i := 0; i < 100; i++ {
		a := gen(Address{}, r).(Address)
		addresses = append(addresses, a)
		err = kad.AddNode(&testNode{addr: a})
		if err != nil {
			t.Errorf("backend not accepting node")
			return
		}
	}
	for _, a := range addresses {
		called = make(chan bool)
		kad.RemoveNode(&testNode{a})
		b := kad.proximityBin(a)
		select {
		case <-called:
			if bucket != b {
				t.Errorf("GetNode callback called with incorrect bucket, expected %v, got %v", b, bucket)
			}
		case <-time.After(100 * time.Millisecond):
			l := len(kad.buckets[b].nodes)
			if l < kad.BucketSize {
				t.Errorf("GetNode not called on bucket %v although size is %v < %v", b, l, kad.BucketSize)
			} else {
				t.Logf("bucket callback ok")
			}
		}
	}

}

func TestSaveLoad(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	addresses := gen([]Address{}, r).([]Address)
	self := addresses[0]
	kad := New(self)
	kad.MaxProx = 10
	kad.Start()
	var err error
	for _, a := range addresses[1:] {
		err = kad.AddNode(&testNode{addr: a})
		if err != nil {
			t.Errorf("backend not accepting node")
			return
		}
	}
	nodes := kad.GetNodes(self, 100)
	path := "/tmp/bzz.peers"
	kad.Stop(path)
	kad = New(self)
	kad.Start()
	kad.Load(path)
	for _, b := range kad.DB() {
		for _, node := range b {
			node.node = &testNode{node.Address}
			err = kad.AddNode(node.node)
			if err != nil {
				t.Errorf("backend not accepting node")
				return
			}
		}
	}
	loadednodes := kad.GetNodes(self, 100)
	for i, node := range loadednodes {
		if nodes[i].Addr() != node.Addr() {
			t.Errorf("node mismatch at %d/%d", i, len(nodes))
		}
	}
}

func (self *Kademlia) proxCheck(t *testing.T) bool {
	var sum, i int
	var b *bucket
	for i, b = range self.buckets {
		l := len(b.nodes)
		// if we are in the high prox multibucket
		if i >= self.proxLimit {
			// unless it starts with an empty bucket, count the size
			if l > 0 || sum > 0 {
				sum += l
			}
		} else if l == 0 {
			t.Errorf("bucket %d empty, yet proxLimit is %d", len(b.nodes), self.proxLimit)
			return false
		}
	}
	// check if merged high prox bucket does not exceed size
	if sum > 0 {
		if sum > self.MaxProxBinSize {
			t.Errorf("bucket %d is empty, yet proxSize is %d", i, self.proxSize)
			return false
		}
		if sum != self.proxSize {
			t.Errorf("proxSize incorrect, expected %v, got %v", sum, self.proxSize)
			return false
		}
	}
	return true
}

type getNodesTest struct {
	Self   Address
	Target Address
	All    []Node
	N      int
}

func (c getNodesTest) String() string {
	return fmt.Sprintf("A: %x\nT: %x\n(%d)\n", c.Self, c.Target, c.N)
}

func (*getNodesTest) Generate(rand *rand.Rand, size int) reflect.Value {
	t := &getNodesTest{
		Self:   gen(Address{}, rand).(Address),
		Target: gen(Address{}, rand).(Address),
		N:      rand.Intn(bucketSize),
	}
	for _, a := range gen([]Address{}, rand).([]Address) {
		t.All = append(t.All, &testNode{addr: a})
	}
	return reflect.ValueOf(t)
}

func (*proxTest) Generate(rand *rand.Rand, size int) reflect.Value {
	var add bool
	if rand.Intn(1) == 0 {
		add = true
	}
	var t *proxTest
	if add {
		t = &proxTest{
			address: gen(Address{}, rand).(Address),
			add:     add,
		}
	} else {
		t = &proxTest{
			index: rand.Intn(len(addresses)),
			add:   add,
		}
	}
	return reflect.ValueOf(t)
}

func hasDuplicates(slice []Node) bool {
	seen := make(map[Address]bool)
	for _, node := range slice {
		if seen[node.Addr()] {
			return true
		}
		seen[node.Addr()] = true
	}
	return false
}

func contains(nodes []Node, addr Address) bool {
	for _, n := range nodes {
		if n.Addr() == addr {
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

func (Address) Generate(rand *rand.Rand, size int) reflect.Value {
	var id Address
	// m := rand.Intn(len(id))
	for i := 0; i < len(id); i++ {
		id[i] = byte(rand.Uint32())
	}
	return reflect.ValueOf(id)
}
