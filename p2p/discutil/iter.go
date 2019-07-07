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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
)

// Iterator represents a sequence of nodes.
//
// The NextNode method returns the next node in the sequence. It may return nil when no
// node could be found before the context was canceled. The isLive return value reports
// whether the iterator is still open. Once closed, iterators should keep returning (nil, false).
//
// Implementations of NextNode are not required to be safe for concurrent use. It is
// therefore unsafe to call NextNode from multiple goroutines at the same time.
//
// Close may be called concurrently with NextNode, and interrupts NextNode.
type Iterator interface {
	NextNode(ctx context.Context) (n *enode.Node, isLive bool)
	Close()
}

// ReadNodes reads at most n nodes from the given iterator. The return value contains no
// duplicates and no nil values. To prevent looping indefinitely for small repeating node
// sequences, this function calls NextNode at most n times.
func ReadNodes(ctx context.Context, it Iterator, n int) []*enode.Node {
	seen := make(map[enode.ID]*enode.Node, n)
	for i := 0; i < n && ctx.Err() == nil; i++ {
		node, isLive := it.NextNode(ctx)
		if !isLive {
			break
		}
		if node == nil {
			continue
		}
		// Remove duplicates, keeping the node with higher seq.
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

// Filter wraps an iterator such that NextNode only returns nodes for which
// the 'check' function returns true.
func Filter(it Iterator, check func(*enode.Node) bool) Iterator {
	return &filterIter{it, check}
}

type filterIter struct {
	it    Iterator
	check func(*enode.Node) bool
}

func (f *filterIter) NextNode(ctx context.Context) (*enode.Node, bool) {
	n, isLive := f.it.NextNode(ctx)
	if n != nil && !f.check(n) {
		n = nil
	}
	return n, isLive
}

func (f *filterIter) Close() {
	f.it.Close()
}

// FairMix aggregates multiple node iterators. The mixer itself is an iterator which ends
// only when Close is called. Source iterators added via AddSource are removed from the mix
// when they end.
//
// The distribution of nodes returned by NextNode is approximately fair, i.e. FairMix
// attempts to draw from all sources equally often. However, if a certain source is slow
// and doesn't return a node within the configured timeout, a node from any other source
// will be returned.
//
// It's safe to call AddSource and Close concurrently with NextNode.
type FairMix struct {
	ctx       context.Context
	cancelCtx func()
	wg        sync.WaitGroup
	fromAny   chan *enode.Node
	timeout   time.Duration

	mu      sync.Mutex
	sources []*mixSource
	last    int
}

type mixSource struct {
	it   Iterator
	next chan *enode.Node
}

// NewFairMix creates a mixer.
//
// The timeout specifies how long the mixer will wait for the next fairly-chosen source
// before giving up and taking a node from any other source. A good way to set the timeout
// is deciding how long you'd want to wait for a node on average. Passing a negative
// timeout disables the mixer completely fair.
func NewFairMix(timeout time.Duration) *FairMix {
	ctx, cancel := context.WithCancel(context.Background())
	m := &FairMix{
		ctx:       ctx,
		cancelCtx: cancel,
		fromAny:   make(chan *enode.Node),
		timeout:   timeout,
	}
	return m
}

// AddSource adds a source of nodes.
func (m *FairMix) AddSource(it Iterator) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isLive() {
		return
	}
	m.wg.Add(1)
	source := &mixSource{it, make(chan *enode.Node)}
	m.sources = append(m.sources, source)
	go m.runSource(source)
}

// Close shuts down the mixer. Calling this is required to release resources
// associated with the mixer.
func (m *FairMix) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isLive() {
		return
	}
	m.cancelCtx()
	m.wg.Wait()
	m.sources = nil
	close(m.fromAny)
}

// NextNode returns a node from a random source.
func (m *FairMix) NextNode(ctx context.Context) (*enode.Node, bool) {
	var timeout <-chan time.Time
	if m.timeout >= 0 {
		timer := time.NewTimer(m.timeout)
		timeout = timer.C
		defer timer.Stop()
	}
	for {
		source := m.pickSource()
		if source == nil {
			return m.nextFromAny(ctx)
		}
		select {
		case n, ok := <-source.next:
			if !ok {
				// This source has ended.
				m.deleteSource(source)
				continue
			}
			return n, m.isLive()
		case <-timeout:
			return m.nextFromAny(ctx)
		case <-ctx.Done():
			return nil, m.isLive()
		}
	}
}

// nextFromAny is used when there are no sources or when the 'fair' choice
// doesn't turn up a node quickly enough.
func (m *FairMix) nextFromAny(ctx context.Context) (*enode.Node, bool) {
	select {
	case n, ok := <-m.fromAny:
		return n, ok
	case <-ctx.Done():
		return nil, m.isLive()
	}
}

func (m *FairMix) isLive() bool {
	return m.ctx.Err() == nil
}

// pickSource chooses the next source to read from, cycling through them in order.
func (m *FairMix) pickSource() *mixSource {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sources) == 0 {
		return nil
	}
	m.last = (m.last + 1) % len(m.sources)
	return m.sources[m.last]
}

// deleteSource deletes a source.
func (m *FairMix) deleteSource(s *mixSource) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.sources {
		if m.sources[i] == s {
			copy(m.sources[i:], m.sources[i+1:])
			m.sources[len(m.sources)-1] = nil
			m.sources = m.sources[:len(m.sources)-1]
			break
		}
	}
}

// runSource reads a single source in a loop.
func (m *FairMix) runSource(s *mixSource) {
	defer m.wg.Done()
	defer close(s.next)
	for {
		n, isLive := s.it.NextNode(m.ctx)
		if !isLive {
			return
		}
		select {
		case s.next <- n:
		case m.fromAny <- n:
		case <-m.ctx.Done():
			return
		}
	}
}
