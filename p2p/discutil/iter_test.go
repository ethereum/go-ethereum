// Copyright 2019 The go-ethereum Authors
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

package discutil

import (
	"encoding/binary"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
)

func TestReadNodes(t *testing.T) {
	nodes := ReadNodes(new(genIter), 10)
	checkNodes(t, nodes, 10)
}

// This test checks that ReadNodes terminates when reading N nodes from an iterator
// which returns less than N nodes in an endless cycle.
func TestReadNodesCycle(t *testing.T) {
	iter := &callCountIter{
		Iterator: cycleNodes(
			testNode(0, 0),
			testNode(1, 0),
			testNode(2, 0),
		),
	}
	nodes := ReadNodes(iter, 10)
	checkNodes(t, nodes, 3)
	if iter.count != 10 {
		t.Fatalf("%d calls to Next, want %d", iter.count, 100)
	}
}

func checkNodes(t *testing.T, nodes []*enode.Node, wantLen int) {
	if len(nodes) != wantLen {
		t.Errorf("slice has %d nodes, want %d", len(nodes), wantLen)
		return
	}
	seen := make(map[enode.ID]bool)
	for i, e := range nodes {
		if e == nil {
			t.Errorf("nil node at index %d", i)
			return
		}
		if seen[e.ID()] {
			t.Errorf("slice has duplicate node %v", e.ID())
			return
		}
		seen[e.ID()] = true
	}
}

// This test checks fairness of FairMix in the happy case where all sources return nodes
// within the context's deadline.
func TestFairMix(t *testing.T) {
	for i := 0; i < 500; i++ {
		testMixerFairness(t)
	}
}

func testMixerFairness(t *testing.T) {
	mix := NewFairMix(1 * time.Second)
	mix.AddSource(&genIter{index: 1})
	mix.AddSource(&genIter{index: 2})
	mix.AddSource(&genIter{index: 3})
	defer mix.Close()

	nodes := ReadNodes(mix, 500)
	checkNodes(t, nodes, 500)

	// Verify that the nodes slice contains an approximately equal number of nodes
	// from each source.
	d := idPrefixDistribution(nodes)
	for _, count := range d {
		if approxEqual(count, len(nodes)/3, 30) {
			t.Fatalf("ID distribution is unfair: %v", d)
		}
	}
}

// This test checks that FairMix falls back to an alternative source when
// the 'fair' choice doesn't return a node within the timeout.
func TestFairMixNextFromAll(t *testing.T) {
	mix := NewFairMix(1 * time.Millisecond)
	mix.AddSource(&genIter{index: 1})
	mix.AddSource(cycleNodes())
	defer mix.Close()

	nodes := ReadNodes(mix, 500)
	checkNodes(t, nodes, 500)

	d := idPrefixDistribution(nodes)
	if len(d) > 1 || d[1] != len(nodes) {
		t.Fatalf("wrong ID distribution: %v", d)
	}
}

// This test ensures FairMix works for Next with no sources.
func TestFairMixEmpty(t *testing.T) {
	var (
		mix   = NewFairMix(1 * time.Second)
		testN = testNode(1, 1)
		ch    = make(chan *enode.Node)
	)
	defer mix.Close()

	go func() {
		mix.Next()
		ch <- mix.Node()
	}()

	mix.AddSource(cycleNodes(testN))
	if n := <-ch; n != testN {
		t.Errorf("got wrong node: %v", n)
	}
}

// This test checks closing a source while Next runs.
func TestFairMixRemoveSource(t *testing.T) {
	mix := NewFairMix(1 * time.Second)
	source := cycleNodes()
	source.Close()
	mix.AddSource(source)

	if mix.Next() {
		t.Fatal("Next should've returned false")
	}
	if len(mix.sources) != 0 {
		t.Fatalf("have %d sources, want zero", len(mix.sources))
	}
}

func TestFairMixClose(t *testing.T) {
	for i := 0; i < 20 && !t.Failed(); i++ {
		testMixerClose(t)
	}
}

func testMixerClose(t *testing.T) {
	mix := NewFairMix(-1)
	mix.AddSource(cycleNodes())
	mix.AddSource(cycleNodes())

	done := make(chan struct{})
	go func() {
		defer close(done)
		if mix.Next() {
			t.Error("Next returned true")
		}
	}()
	// This call is supposed to make it more likely that NextNode is
	// actually executing by the time we call Close.
	runtime.Gosched()

	mix.Close()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Next didn't unblock on Close")
	}

	mix.Close() // shouldn't crash
}

func idPrefixDistribution(nodes []*enode.Node) map[uint32]int {
	d := make(map[uint32]int)
	for _, node := range nodes {
		id := node.ID()
		d[binary.BigEndian.Uint32(id[:4])]++
	}
	return d
}

func approxEqual(x, y, ε int) bool {
	if y > x {
		x, y = y, x
	}
	return x-y > ε
}

// genIter creates fake nodes with numbered IDs based on 'index' and 'gen'
type genIter struct {
	node       *enode.Node
	index, gen uint32
}

func (s *genIter) Next() bool {
	index := atomic.LoadUint32(&s.index)
	if index == ^uint32(0) {
		s.node = nil
		return false
	}
	s.node = testNode(uint64(index)<<32|uint64(s.gen), 0)
	s.gen++
	return true
}

func (s *genIter) Node() *enode.Node {
	return s.node
}

func (s *genIter) Close() {
	s.index = ^uint32(0)
}

func testNode(id, seq uint64) *enode.Node {
	var nodeID enode.ID
	binary.BigEndian.PutUint64(nodeID[:], id)
	r := new(enr.Record)
	r.SetSeq(seq)
	return enode.SignNull(r, nodeID)
}

// cycleNodes is an interator that cycles through the given slice.
func cycleNodes(nodes ...*enode.Node) Iterator {
	return &cycleIter{nodes: nodes}
}

type cycleIter struct {
	cur   *enode.Node
	mu    sync.Mutex
	index int
	nodes []*enode.Node
}

func (s *cycleIter) Next() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.nodes) == 0 {
		return false
	}
	s.cur = s.nodes[s.index]
	s.index = (s.index + 1) % len(s.nodes)
	return true
}

func (s *cycleIter) Node() *enode.Node {
	return s.nodes[s.index]
}

func (s *cycleIter) Close() {
	s.mu.Lock()
	s.nodes = nil
	s.mu.Unlock()
}

// callCountIter counts calls to NextNode.
type callCountIter struct {
	Iterator
	count int
}

func (it *callCountIter) Next() bool {
	it.count++
	return it.Iterator.Next()
}
