// Copyright 2026 The go-ethereum Authors
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

package discover

import (
	"crypto/ecdsa"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
)

// makeTestNodes returns n deterministically-generated nodes so callers can
// build small synthetic graphs.
func makeTestNodes(t *testing.T, n int) []*enode.Node {
	t.Helper()
	nodes := make([]*enode.Node, n)
	for i := 0; i < n; i++ {
		var key *ecdsa.PrivateKey
		var err error
		key, err = crypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		var r enr.Record
		r.Set(enr.IPv4{127, 0, 0, 1})
		r.Set(enr.UDP(30300 + i))
		if err := enode.SignV4(&r, key); err != nil {
			t.Fatal(err)
		}
		nodes[i], err = enode.New(enode.ValidSchemes, &r)
		if err != nil {
			t.Fatal(err)
		}
	}
	return nodes
}

// TestCrawlIteratorTerminates verifies that the iterator emits every node in
// a finite synthetic graph, exactly once, and then returns false from Next.
func TestCrawlIteratorTerminates(t *testing.T) {
	nodes := makeTestNodes(t, 50)

	// Build a synthetic neighbour map: each node knows the next 5 nodes
	// (cyclic). The crawl should reach all 50 from any single seed.
	neighbours := make(map[enode.ID][]*enode.Node, len(nodes))
	for i, n := range nodes {
		var ns []*enode.Node
		for k := 1; k <= 5; k++ {
			ns = append(ns, nodes[(i+k)%len(nodes)])
		}
		neighbours[n.ID()] = ns
	}

	var calls atomic.Int64
	queryFn := func(dst *enode.Node, _ int) ([]*enode.Node, error) {
		calls.Add(1)
		return neighbours[dst.ID()], nil
	}

	it := newCrawlIterator(CrawlOptions{
		Workers: 4,
		Seeds:   []*enode.Node{nodes[0]},
		Drange:  16,
	}, queryFn)

	seen := make(map[enode.ID]int)
	done := make(chan struct{})
	go func() {
		for it.Next() {
			seen[it.Node().ID()]++
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("iterator did not terminate within 5s")
	}

	if got := len(seen); got != len(nodes) {
		t.Fatalf("emitted %d distinct nodes, want %d", got, len(nodes))
	}
	for id, c := range seen {
		if c != 1 {
			t.Errorf("node %x emitted %d times, want 1", id[:4], c)
		}
	}
	// Every distinct node should have been queried once.
	if got, want := calls.Load(), int64(len(nodes)); got != want {
		t.Errorf("queryFn invoked %d times, want %d", got, want)
	}
}

// TestCrawlIteratorClose verifies that calling Close while the iterator is
// still discovering nodes unblocks Next and stops workers cleanly.
func TestCrawlIteratorClose(t *testing.T) {
	nodes := makeTestNodes(t, 20)

	// Slow queryFn so we can interrupt mid-crawl.
	queryFn := func(dst *enode.Node, _ int) ([]*enode.Node, error) {
		time.Sleep(50 * time.Millisecond)
		var ns []*enode.Node
		for i, n := range nodes {
			if n.ID() == dst.ID() {
				ns = append(ns, nodes[(i+1)%len(nodes)])
				break
			}
		}
		return ns, nil
	}

	it := newCrawlIterator(CrawlOptions{
		Workers: 2,
		Seeds:   []*enode.Node{nodes[0]},
		Drange:  16,
	}, queryFn)

	// Drain a few nodes, then Close.
	go func() {
		time.Sleep(50 * time.Millisecond)
		it.Close()
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	done := make(chan struct{})
	go func() {
		defer wg.Done()
		for it.Next() {
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Next did not return after Close")
	}
	wg.Wait()
}

// TestCrawlIteratorOutputCap verifies the backpressure invariant:
// the size of the output buffer never exceeds OutputCap regardless of
// how fast the queryFn returns peers.
func TestCrawlIteratorOutputCap(t *testing.T) {
	const cap = 8
	nodes := makeTestNodes(t, 200)

	// Each node maps to the next in the chain, so discovery is unbounded
	// from the iterator's perspective until we've covered the cycle.
	queryFn := func(dst *enode.Node, _ int) ([]*enode.Node, error) {
		for i, n := range nodes {
			if n.ID() == dst.ID() {
				// Return 4 fresh neighbours so the producer outpaces the
				// consumer (we sleep between Next() calls below).
				return []*enode.Node{
					nodes[(i+1)%len(nodes)],
					nodes[(i+2)%len(nodes)],
					nodes[(i+3)%len(nodes)],
					nodes[(i+4)%len(nodes)],
				}, nil
			}
		}
		return nil, nil
	}

	itAny := newCrawlIterator(CrawlOptions{
		Workers:   8,
		Seeds:     []*enode.Node{nodes[0]},
		Drange:    16,
		OutputCap: cap,
	}, queryFn)
	it := itAny // *crawlIterator

	var maxObserved int
	check := func() {
		it.mu.Lock()
		if l := len(it.output); l > maxObserved {
			maxObserved = l
		}
		it.mu.Unlock()
	}

	// Slow consumer: read with delays so workers must back off.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for it.Next() {
			check()
			time.Sleep(2 * time.Millisecond)
		}
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		it.Close()
		t.Fatal("iterator did not terminate within 10s")
	}

	if maxObserved > cap {
		t.Errorf("output buffer reached %d, want <= cap=%d", maxObserved, cap)
	}
}

// TestCrawlIteratorRotation verifies that the d argument passed to queryFn
// rotates through 0..Drange-1.
func TestCrawlIteratorRotation(t *testing.T) {
	nodes := makeTestNodes(t, 64)
	// Each node has the next 1 as neighbour, so the crawl makes exactly
	// len(nodes) FINDNODE calls in a chain.
	neighbours := make(map[enode.ID]*enode.Node, len(nodes))
	for i, n := range nodes {
		neighbours[n.ID()] = nodes[(i+1)%len(nodes)]
	}

	var (
		mu      sync.Mutex
		seenDs  = make(map[int]int)
	)
	queryFn := func(dst *enode.Node, d int) ([]*enode.Node, error) {
		mu.Lock()
		seenDs[d]++
		mu.Unlock()
		return []*enode.Node{neighbours[dst.ID()]}, nil
	}

	it := newCrawlIterator(CrawlOptions{
		Workers: 1, // single worker → strictly increasing d
		Seeds:   []*enode.Node{nodes[0]},
		Drange:  16,
	}, queryFn)
	for it.Next() {
	}

	mu.Lock()
	defer mu.Unlock()
	for d := 0; d < 16; d++ {
		if seenDs[d] == 0 {
			t.Errorf("rotation index %d never used", d)
		}
	}
}
