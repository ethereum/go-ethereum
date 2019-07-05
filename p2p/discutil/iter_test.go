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
	"context"
	"encoding/binary"
	"runtime"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
)

func TestReadNodes(t *testing.T) {
	iter := new(genIter)
	nodes := ReadNodes(context.Background(), iter, 10)
	checkNodes(t, nodes, 10)
}

// This test verifies that ReadNodes checks for context cancelation.
func TestReadNodesCancel(t *testing.T) {
	iter := &blockedIter{new(genIter), nil}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	nodes := ReadNodes(ctx, iter, 10)
	checkNodes(t, nodes, 0)
}

// // This test checks that ReadNodes terminates when reading N nodes from an iterator
// // which returns less than N nodes in an endless cycle.
func TestReadNodesCycle(t *testing.T) {
	iter := &callCountIter{
		child: cycleNodes{
			testNode(0, 0),
			testNode(1, 0),
			testNode(2, 0),
		},
	}
	nodes := ReadNodes(context.Background(), iter, 10)
	checkNodes(t, nodes, 3)
	if iter.count != 10 {
		t.Fatalf("%d calls to NextNode, want %d", iter.count, 100)
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

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	nodes := ReadNodes(ctx, mix, 500)
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
// the 'fair' choice doesn't return a node within the context's deadline.
func TestFairMixNextFromAll(t *testing.T) {
	mix := NewFairMix(1 * time.Millisecond)
	mix.AddSource(&genIter{index: 1})
	mix.AddSource(&blockedIter{child: &genIter{index: 2}})
	defer mix.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	nodes := ReadNodes(ctx, mix, 500)
	checkNodes(t, nodes, 500)

	d := idPrefixDistribution(nodes)
	if len(d) > 1 || d[1] != len(nodes) {
		t.Fatalf("wrong ID distribution: %v", d)
	}
}

// This test ensures FairMix works for NextNode with no sources.
func TestFairMixEmpty(t *testing.T) {
	var (
		mix   = NewFairMix(1 * time.Second)
		testN = testNode(1, 1)
		ch    = make(chan *enode.Node)
	)
	defer mix.Close()

	go func() {
		n, _ := mix.NextNode(context.Background())
		ch <- n
	}()

	mix.AddSource(cycleNodes{testN})
	if n := <-ch; n != testN {
		t.Errorf("got wrong node: %v", n)
	}
}

// This test checks closing a source while NextNode runs.
func TestFairMixRemoveSource(t *testing.T) {
	mix := NewFairMix(1 * time.Second)
	source := &blockedIter{child: &genIter{index: 1}, unblock: make(chan struct{})}
	close(source.unblock) // first NextNode call will return (nil, false)
	mix.AddSource(source)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	n, isLive := mix.NextNode(ctx)
	if n != nil {
		t.Fatal("NextNode returned a node but shouldn't")
	}
	if !isLive {
		t.Fatal("NextNode returned isLive == false")
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
	mix.AddSource(cycleNodes{})
	mix.AddSource(cycleNodes{})

	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, isLive := mix.NextNode(context.Background()); isLive {
			t.Error("NextNode returned isLive == true")
		}
	}()
	// This call is supposed to make it more likely that NextNode is
	// actually executing by the time we call Close.
	runtime.Gosched()

	mix.Close()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("NextNode didn't unblock on Close")
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
	index, gen uint32
}

func (s *genIter) NextNode(ctx context.Context) (*enode.Node, bool) {
	n := testNode(uint64(s.index)<<32|uint64(s.gen), 0)
	s.gen++
	return n, true
}

func testNode(id, seq uint64) *enode.Node {
	var nodeID enode.ID
	binary.BigEndian.PutUint64(nodeID[:], id)
	r := new(enr.Record)
	r.SetSeq(seq)
	return enode.SignNull(r, nodeID)
}

// blockedIter delays NextNodes until the unblock channel receives a value.
type blockedIter struct {
	child   Iterator
	unblock chan struct{}
}

func (s *blockedIter) NextNode(ctx context.Context) (*enode.Node, bool) {
	select {
	case _, ok := <-s.unblock:
		if !ok {
			return nil, false
		}
		return s.child.NextNode(ctx)
	case <-ctx.Done():
		return nil, true
	}
}

// cycleNodes is a never-ending interator that cycles through the given slice.
type cycleNodes []*enode.Node

func (s cycleNodes) NextNode(ctx context.Context) (*enode.Node, bool) {
	if len(s) == 0 {
		<-ctx.Done()
		return nil, true
	}
	n := s[0]
	copy(s[:], s[1:])
	s[len(s)-1] = n
	return n, true
}

// callCountIter counts calls to NextNode.
type callCountIter struct {
	child Iterator
	count int
}

func (it *callCountIter) NextNode(ctx context.Context) (*enode.Node, bool) {
	it.count++
	return it.child.NextNode(ctx)
}
