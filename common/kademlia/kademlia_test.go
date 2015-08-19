package kademlia

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
	"time"
)

var (
	quickrand         = rand.New(rand.NewSource(time.Now().Unix()))
	quickcfgGetNodes  = &quick.Config{MaxCount: 5000, Rand: quickrand}
	quickcfgBootStrap = &quick.Config{MaxCount: 1000, Rand: quickrand}
)

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
	addr, ok := gen(Address{}, quickrand).(Address)
	other, ok := gen(Address{}, quickrand).(Address)
	if !ok {
		t.Errorf("oops")
	}
	kad := New()
	kad.Start(addr)
	err := kad.AddNode(&testNode{addr: other})
	_ = err
}

func TestBootstrap(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	test := func(test *bootstrapTest) bool {
		// for any node kad.le, Target and N
		kad := New()
		kad.MaxProx = test.MaxProx
		kad.BucketSize = test.BucketSize
		kad.Start(test.Self)
		var err error

		// t.Logf("bootstapTest MaxProx: %v BucketSize: %v\n", test.MaxProx, test.BucketSize)

		addr := gen(Address{}, r).(Address)
		prox := proximity(addr, test.Self)

		for p := 0; p <= prox; p++ {
			var nrs []*NodeRecord
			for i := 0; i < test.BucketSize; i++ {
				nrs = append(nrs, &NodeRecord{
					Addr: RandomAddressAt(test.Self, p),
				})
			}
			kad.AddNodeRecords(nrs)
		}

		node := &testNode{addr}

		n := 0
		for n < 100 {
			err = kad.AddNode(node)
			if err != nil {
				t.Errorf("backend not accepting node")
				return false
			}
			var nrs []*NodeRecord
			prox := proximity(node.addr, test.Self)
			for i := 0; i < test.BucketSize; i++ {
				nrs = append(nrs, &NodeRecord{
					Addr: RandomAddressAt(test.Self, prox+1),
				})
			}
			kad.AddNodeRecords(nrs)

			var lens []int
			for i := 0; i <= test.MaxProx; i++ {
				lens = append(lens, len(kad.buckets[i].nodes))
			}

			record, _ := kad.GetNodeRecord()
			if record == nil {
				// t.Logf("after round %d, no more node records needed", n)
				break
			}
			node = &testNode{record.Addr}
			n++
		}
		exp := test.BucketSize * (test.MaxProx + 1)
		if kad.Count() != exp {
			t.Errorf("incorrect number of peers, expected %d, got %d\n%v", exp, kad.Count(), kad)
			return false
		}
		return true
	}
	if err := quick.Check(test, quickcfgBootStrap); err != nil {
		t.Error(err)
	}

}

func TestGetNodes(t *testing.T) {
	t.Parallel()

	test := func(test *getNodesTest) bool {
		// for any node kad.le, Target and N
		kad := New()
		kad.MaxProx = 10
		kad.Start(test.Self)
		var err error
		// t.Logf("getNodesTest %v: %v\n", len(test.All), test)
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
				if test.Target.ProxCmp(n.Addr(), farthestResult) < 0 {
					_ = i * j
					t.Errorf("kad.le contains node that is closer to target but it's not in result")
					// t.Logf("bucket %v, item %v\n", i, j)
					// t.Logf("  Target:          %x", test.Target)
					// t.Logf("  Farthest Result: %x", farthestResult)
					// t.Logf("  ID:              %x (%d)", n.Addr(), kad.proximityBin(n.Addr()))
					return false
				}
			}
		}
		return true
	}
	if err := quick.Check(test, quickcfgGetNodes); err != nil {
		t.Error(err)
	}
}

type proxTest struct {
	add   bool
	index int
	addr  Address
}

var (
	addresses []Address
)

func TestProxAdjust(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	self := gen(Address{}, r).(Address)

	kad := New()
	kad.MaxProx = 10
	kad.Start(self)
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
		node := &testNode{test.addr}
		if test.add {
			kad.AddNode(node)
		} else {
			kad.RemoveNode(node)
		}
		return kad.proxCheck(t)
	}
	if err := quick.Check(test, quickcfgGetNodes); err != nil {
		t.Error(err)
	}
}

func TestSaveLoad(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	addresses := gen([]Address{}, r).([]Address)
	self := addresses[0]
	kad := New()
	kad.MaxProx = 10
	kad.Start(self)
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
	kad = New()
	kad.Start(self)
	kad.Load(path)
	for _, b := range kad.nodeDB {
		for _, node := range b {
			node.node = &testNode{node.Addr}
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
			sum += l
		} else if l == 0 {
			t.Errorf("bucket %d empty, yet proxLimit is %d\n%v", len(b.nodes), self.proxLimit, self)
			return false
		}
	}
	// check if merged high prox bucket does not exceed size
	if sum > 0 {
		// if sum > self.ProxBinSize {
		// 	t.Errorf("bucket %d is empty, yet proxSize is %d\n%v", i, self.proxSize, self)
		// 	return false
		// }
		if sum != self.proxSize {
			t.Errorf("proxSize incorrect, expected %v, got %v", sum, self.proxSize)
			return false
		}
		if self.proxLimit > 0 && sum+len(self.buckets[self.proxLimit-1].nodes) < self.ProxBinSize {
			t.Errorf("proxBinSize incorrect, expected %v got %v", sum, self.proxSize)
			return false
		}
	}
	return true
}

type bootstrapTest struct {
	MaxProx    int
	BucketSize int
	Self       Address
}

func (*bootstrapTest) Generate(rand *rand.Rand, size int) reflect.Value {
	t := &bootstrapTest{
		Self:       gen(Address{}, rand).(Address),
		MaxProx:    10 + rand.Intn(3),
		BucketSize: rand.Intn(3) + 1,
	}
	return reflect.ValueOf(t)
}

type getNodesTest struct {
	Self   Address
	Target Address
	All    []Node
	N      int
}

func (c getNodesTest) String() string {
	return fmt.Sprintf("A: %064x\nT: %064x\n(%d)\n", c.Self[:], c.Target[:], c.N)
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
			addr: gen(Address{}, rand).(Address),
			add:  add,
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
