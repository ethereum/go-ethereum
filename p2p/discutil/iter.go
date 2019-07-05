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

// Package discutil provides node discovery utilities.
package discutil

import (
	"context"

	"github.com/ethereum/go-ethereum/p2p/enode"
)

// Iterator represents an infinite sequence of nodes. The NextNode method returns the next
// node in the sequence. It may return nil if no next node could be found before the
// context was canceled. Implementations are not required to be safe for concurrent use.
type Iterator interface {
	NextNode(ctx context.Context) *enode.Node
}

// ReadNodes reads at most n nodes from the given iterator. The returned slice contains no
// duplicates and no nil values. To prevent looping indefinitely for small repeating node
// sequences, e.g. when reading from a CycleNodes iterator with a slice length < n, this
// function calls NextNode at most n times.
func ReadNodes(ctx context.Context, it Iterator, n int) []*enode.Node {
	seen := make(map[enode.ID]*enode.Node, n)
	for i := 0; i < n && ctx.Err() == nil; i++ {
		node := it.NextNode(ctx)
		if node == nil {
			continue
		}
		prevNode, ok := seen[node.ID()]
		if ok && prevNode.Seq() > node.Seq() {
			continue
		}
		seen[node.ID()] = node
	}
	result := make([]*enode.Node, 0, len(seen))
	for _, node := range seen {
		result = append(result, node)
	}
	return result
}

// CycleNodes returns a never-ending interator that cycles through the given slice.
func CycleNodes(nodes []*enode.Node) Iterator {
	if len(nodes) == 0 {
		return IterFunc(nullIterator)
	}
	index := 0
	return IterFunc(func(context.Context) *enode.Node {
		n := nodes[index]
		index = (index + 1) % len(nodes)
		return n
	})
}

func nullIterator(context.Context) *enode.Node {
	return nil
}

// IterChan returns a NodeIterator wrapping the given channel.
func IterChan(ch <-chan *enode.Node) Iterator {
	return IterFunc(func(ctx context.Context) *enode.Node {
		select {
		case n := <-ch:
			return n
		case <-ctx.Done():
			return nil
		}
	})
}

// IterFunc is a function that satisfies the NodeIterator interface.
type IterFunc func(ctx context.Context) *enode.Node

// NextNode calls the function.
func (fn IterFunc) NextNode(ctx context.Context) *enode.Node {
	return fn(ctx)
}

// Mixer aggregates multiple node iterators. The distribution of nodes drawn from the mixer
// is fair, i.e. all iterators are drawn from equally often.
type Mixer struct {
	sources []Iterator
	last    int
}

// NewMixer creates a Mixer with the given initial sources.
func NewMixer(sources ...Iterator) *Mixer {
	return &Mixer{sources: sources}
}

// AddSource adds a source of nodes.
func (m *Mixer) AddSource(source Iterator) {
	m.sources = append(m.sources, source)
}

// NextNode returns a node from a random source.
func (m *Mixer) NextNode(ctx context.Context) *enode.Node {
	if len(m.sources) == 0 {
		return nil
	}
	source := m.nextSource()
	return source.NextNode(ctx)
}

func (m *Mixer) nextSource() Iterator {
	s := m.sources[m.last]
	m.last = (m.last + 1) % len(m.sources)
	return s
}
