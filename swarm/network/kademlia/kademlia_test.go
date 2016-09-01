// Copyright 2016 The go-ethereum Authors
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

package kademlia

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"testing/quick"
	"time"
)

var (
	quickrand           = rand.New(rand.NewSource(time.Now().Unix()))
	quickcfgFindClosest = &quick.Config{MaxCount: 50, Rand: quickrand}
	quickcfgBootStrap   = &quick.Config{MaxCount: 100, Rand: quickrand}
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

func TestOn(t *testing.T) {
	addr, ok := gen(Address{}, quickrand).(Address)
	other, ok := gen(Address{}, quickrand).(Address)
	if !ok {
		t.Errorf("oops")
	}
	kad := New(addr, NewKadParams())
	err := kad.On(&testNode{addr: other}, nil)
	_ = err
}

func TestBootstrap(t *testing.T) {

	test := func(test *bootstrapTest) bool {
		// for any node kad.le, Target and N
		params := NewKadParams()
		params.MaxProx = test.MaxProx
		params.BucketSize = test.BucketSize
		params.ProxBinSize = test.BucketSize
		kad := New(test.Self, params)
		var err error

		for p := 0; p < 9; p++ {
			var nrs []*NodeRecord
			n := math.Pow(float64(2), float64(7-p))
			for i := 0; i < int(n); i++ {
				addr := RandomAddressAt(test.Self, p)
				nrs = append(nrs, &NodeRecord{
					Addr: addr,
				})
			}
			kad.Add(nrs)
		}

		node := &testNode{test.Self}

		n := 0
		for n < 100 {
			err = kad.On(node, nil)
			if err != nil {
				t.Fatalf("backend not accepting node: %v", err)
			}

			record, need, _ := kad.Suggest()
			if !need {
				break
			}
			n++
			if record == nil {
				continue
			}
			node = &testNode{record.Addr}
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

func TestFindClosest(t *testing.T) {

	test := func(test *FindClosestTest) bool {
		// for any node kad.le, Target and N
		params := NewKadParams()
		params.MaxProx = 7
		kad := New(test.Self, params)
		var err error
		for _, node := range test.All {
			err = kad.On(node, nil)
			if err != nil && err.Error() != "bucket full" {
				t.Fatalf("backend not accepting node: %v", err)
			}
		}

		if len(test.All) == 0 || test.N == 0 {
			return true
		}
		nodes := kad.FindClosest(test.Target, test.N)

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
			for j, n := range b {
				if contains(nodes, n.Addr()) {
					continue // don't run the check below for nodes in result
				}
				if test.Target.ProxCmp(n.Addr(), farthestResult) < 0 {
					_ = i * j
					t.Errorf("kad.le contains node that is closer to target but it's not in result")
					return false
				}
			}
		}
		return true
	}
	if err := quick.Check(test, quickcfgFindClosest); err != nil {
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
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	self := gen(Address{}, r).(Address)
	params := NewKadParams()
	params.MaxProx = 7
	kad := New(self, params)

	var err error
	for i := 0; i < 100; i++ {
		a := gen(Address{}, r).(Address)
		addresses = append(addresses, a)
		err = kad.On(&testNode{addr: a}, nil)
		if err != nil && err.Error() != "bucket full" {
			t.Fatalf("backend not accepting node: %v", err)
		}
		if !kad.proxCheck(t) {
			return
		}
	}
	test := func(test *proxTest) bool {
		node := &testNode{test.addr}
		if test.add {
			kad.On(node, nil)
		} else {
			kad.Off(node, nil)
		}
		return kad.proxCheck(t)
	}
	if err := quick.Check(test, quickcfgFindClosest); err != nil {
		t.Error(err)
	}
}

func TestSaveLoad(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	addresses := gen([]Address{}, r).([]Address)
	self := RandomAddress()
	params := NewKadParams()
	params.MaxProx = 7
	kad := New(self, params)

	var err error

	for _, a := range addresses {
		err = kad.On(&testNode{addr: a}, nil)
		if err != nil && err.Error() != "bucket full" {
			t.Fatalf("backend not accepting node: %v", err)
		}
	}
	nodes := kad.FindClosest(self, 100)

	path := filepath.Join(os.TempDir(), "bzz-kad-test-save-load.peers")
	err = kad.Save(path, nil)
	if err != nil && err.Error() != "bucket full" {
		t.Fatalf("unepected error saving kaddb: %v", err)
	}
	kad = New(self, params)
	err = kad.Load(path, nil)
	if err != nil && err.Error() != "bucket full" {
		t.Fatalf("unepected error loading kaddb: %v", err)
	}
	for _, b := range kad.db.Nodes {
		for _, node := range b {
			err = kad.On(&testNode{node.Addr}, nil)
			if err != nil && err.Error() != "bucket full" {
				t.Fatalf("backend not accepting node: %v", err)
			}
		}
	}
	loadednodes := kad.FindClosest(self, 100)
	for i, node := range loadednodes {
		if nodes[i].Addr() != node.Addr() {
			t.Errorf("node mismatch at %d/%d: %v != %v", i, len(nodes), nodes[i].Addr(), node.Addr())
		}
	}
}

func (self *Kademlia) proxCheck(t *testing.T) bool {
	var sum int
	for i, b := range self.buckets {
		l := len(b)
		// if we are in the high prox multibucket
		if i >= self.proxLimit {
			sum += l
		} else if l == 0 {
			t.Errorf("bucket %d empty, yet proxLimit is %d\n%v", len(b), self.proxLimit, self)
			return false
		}
	}
	// check if merged high prox bucket does not exceed size
	if sum > 0 {
		if sum != self.proxSize {
			t.Errorf("proxSize incorrect, expected %v, got %v", sum, self.proxSize)
			return false
		}
		last := len(self.buckets[self.proxLimit])
		if last > 0 && sum >= self.ProxBinSize+last {
			t.Errorf("proxLimit %v incorrect, redundant non-empty bucket %d added to proxBin with %v (target %v)\n%v", self.proxLimit, last, sum-last, self.ProxBinSize, self)
			return false
		}
		if self.proxLimit > 0 && sum < self.ProxBinSize {
			t.Errorf("proxLimit %v incorrect. proxSize %v is less than target %v, yet there is more peers", self.proxLimit, sum, self.ProxBinSize)
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
		MaxProx:    5 + rand.Intn(2),
		BucketSize: rand.Intn(3) + 1,
	}
	return reflect.ValueOf(t)
}

type FindClosestTest struct {
	Self   Address
	Target Address
	All    []Node
	N      int
}

func (c FindClosestTest) String() string {
	return fmt.Sprintf("A: %064x\nT: %064x\n(%d)\n", c.Self[:], c.Target[:], c.N)
}

func (*FindClosestTest) Generate(rand *rand.Rand, size int) reflect.Value {
	t := &FindClosestTest{
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
