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
	"testing"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
)

func TestReadNodes(t *testing.T) {
	iter := new(genSource)
	nodes := ReadNodes(context.Background(), iter, 10)
	checkNodes(t, nodes, 10)
}

// This test verifies that ReadNodes checks for context cancelation.
func TestReadNodesCancel(t *testing.T) {
	iter := &blockedIter{new(genSource), nil}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	nodes := ReadNodes(ctx, iter, 10)
	checkNodes(t, nodes, 0)
}

// This test checks that ReadNodes terminates when reading N nodes from an iterator
// which returns less than N nodes in an endless cycle.
func TestReadNodesCycle(t *testing.T) {
	iter := &callCountIter{
		child: CycleNodes([]*enode.Node{
			testNode(0, 0),
			testNode(1, 0),
			testNode(2, 0),
		}),
	}
	nodes := ReadNodes(context.Background(), iter, 10)
	checkNodes(t, nodes, 3)
	if iter.count != 100 {
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

type callCountIter struct {
	child Iterator
	count int
}

func (it *callCountIter) NextNode(ctx context.Context) *enode.Node {
	it.count++
	return it.child.NextNode(ctx)
}

// This test ensures Mixer doesn't crash for NextNode with no sources.
func TestMixerEmpty(t *testing.T) {
	mix := NewMixer()
	_ = mix.NextNode(context.Background())
}

// This test checks fairness of Mixer for the simple case of three non-overlapping sources
// which return nodes immediately.
func TestMixerFairSimple(t *testing.T) {
	sources := []Iterator{&genSource{index: 1}, &genSource{index: 2}, &genSource{index: 3}}
	mix := NewMixer(sources...)
	nodes := ReadNodes(context.Background(), mix, 198)
	if len(nodes) != 198 {
		t.Fatal("wrong count from ReadNodes:", len(nodes), "want:", 198)
	}

	// Compute distribution.
	d := make(map[uint32]int)
	for i, node := range nodes {
		if node == nil {
			t.Fatalf("node %d is nil", i)
		}
		id := node.ID()
		d[binary.BigEndian.Uint32(id[:4])]++
	}
	// Verify that the nodes slice contains an equal number of nodes from each source.
	for _, count := range d {
		if count != len(nodes)/len(sources) {
			t.Fatalf("ID distribution is unfair: %v", d)
		}
	}
}

// genSource creates fake nodes with numbered IDs based on 'index' and 'gen'
type genSource struct {
	index, gen uint32
}

func (s *genSource) NextNode(ctx context.Context) *enode.Node {
	n := testNode(uint64(s.index)<<32|uint64(s.gen), 0)
	s.gen++
	return n
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

func (s *blockedIter) NextNode(ctx context.Context) *enode.Node {
	select {
	case <-s.unblock:
		return s.child.NextNode(ctx)
	case <-ctx.Done():
		return nil
	}
}
