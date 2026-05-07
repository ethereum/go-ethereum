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
	crand "crypto/rand"
	"math/rand/v2"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/p2p/discover/v4wire"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

// CrawlIterator performs a breadth-first crawl of the discv4/discv5
// FINDNODE-response graph. Each worker pops a peer from the work queue,
// issues one FINDNODE call against it, and feeds any newly-seen peers
// back into the queue (and the iterator's output buffer).
//
// BFS (FIFO queue) was chosen over DFS or target-directed lookup for two
// reasons that align with the iterator's intended use case (survey-style
// crawls — devp2p crawl, geth dial-candidate discovery):
//
//   - Coverage: BFS reaches every peer within N hops of the seeds in
//     order, so a time-bounded run produces a representative sample of
//     the reachable graph rather than a deep tendril through one
//     sub-region.
//   - Adversarial resilience: a peer returning malicious "neighbour"
//     claims, dead-end peers, or eclipse-style sub-graphs cannot
//     monopolise the worker pool, because pending work from other
//     branches sits ahead of the attacker's responses in the queue.
//     DFS would amplify each of these attacks.
//
// CrawlIterator is NOT a target-directed search. To find a specific node
// (or the K closest to a known target), use the lookup-based iterator
// returned by [UDPv4.RandomNodes] / [UDPv5.RandomNodes] instead — those
// run an alpha-bounded Kademlia lookup, which is the right shape for
// "find peer X" but the wrong shape for "survey the network".
//
// Pop policy can be tuned via [CrawlOptions.RandomWorkers]: any subset
// of the worker pool can be configured to pop a uniform-random queue
// item instead of the FIFO front, breaking strict ordering as a mild
// anti-fingerprint defence. The remaining workers pop FIFO and preserve
// the BFS character. See the field doc for details.

// CrawlOptions configures a CrawlIterator.
type CrawlOptions struct {
	// Workers is the number of concurrent FINDNODE calls in flight.
	// If <= 0, a default of 16 is used.
	Workers int

	// Seeds are the nodes to start the crawl from. If empty, the iterator
	// terminates immediately. Callers should pass at least the bootnodes.
	Seeds []*enode.Node

	// Drange is the number of FINDNODE rotation slots per peer. Has effect
	// only on the discv5 path, where each rotation slot d maps to the
	// distance value 256-d (so Drange=16 covers distances 256, 255, ..., 241).
	// On the discv4 path it has no effect: targets are random NodeIDs and
	// the rotation counter is unused.
	//
	// Defaults to 16. Capped at 256.
	Drange int

	// OutputCap bounds the number of newly-discovered peers buffered for the
	// caller's Next() to drain. When the buffer reaches this size, workers
	// pause discovery via cond.Wait until Next() drains it. This is the
	// iterator's backpressure point for slow consumers and adversarial peers
	// flooding fresh ENRs in FINDNODE responses.
	//
	// If <= 0, defaults to 16 * Workers (≥ one full FINDNODE response per
	// worker). Set higher for callers that drain in large batches.
	//
	// Note: the dedup set and the worker work-queue are not bounded; their
	// growth is implicit in any iterator that emits each unique peer exactly
	// once across a long crawl. Realistic bound is the reachable DHT size
	// (~1M peers, ~50 MB).
	OutputCap int

	// RandomWorkers is the number of workers in the pool that pop a uniform-
	// random queue item instead of the FIFO front. The remaining
	// Workers - RandomWorkers workers behave as today, popping front. Total
	// worker count is unchanged at Workers; only the BFS-vs-random split
	// differs.
	//
	// Zero (the struct zero-value) selects the library default of
	// Workers/4 rounded down. Pass a negative value (e.g. -1) to force
	// pure BFS with zero random workers. A positive value > Workers is
	// clamped to Workers (i.e. fully-random pop with no FIFO workers).
	//
	// The default Workers/4 mix preserves the BFS coverage character (cold
	// start is dominated by the FIFO majority) while breaking strict FIFO
	// ordering, a mild anti-fingerprint defence. See the file-level comment
	// for details.
	RandomWorkers int
}

func (o *CrawlOptions) withDefaults() {
	if o.Workers <= 0 {
		o.Workers = 16
	}
	if o.Drange <= 0 {
		o.Drange = 16
	}
	if o.Drange > 256 {
		o.Drange = 256
	}
	if o.OutputCap <= 0 {
		o.OutputCap = 16 * o.Workers
	}
	switch {
	case o.RandomWorkers < 0:
		// Negative means: explicit pure BFS, no random workers.
		o.RandomWorkers = 0
	case o.RandomWorkers == 0:
		// Zero (struct zero-value) means: library default.
		o.RandomWorkers = o.Workers / 4
	case o.RandomWorkers > o.Workers:
		// Clamp; can't have more random workers than total workers.
		o.RandomWorkers = o.Workers
	}
}

// CrawlIterator returns an enode.Iterator that performs a breadth-first
// crawl by issuing a single FINDNODE request per discovered peer, with a
// fresh random target each call. Compared to RandomNodes, this avoids the
// alpha-bounded Kademlia lookup convergence loop and is the right shape
// for breadth crawls (e.g. devp2p discv4 crawl).
//
// Concurrency is bounded by opts.Workers; pacing is RTT-driven, not
// rate-limited.
func (t *UDPv4) CrawlIterator(opts CrawlOptions) enode.Iterator {
	queryFn := func(dst *enode.Node, _ int) ([]*enode.Node, error) {
		addr, ok := dst.UDPEndpoint()
		if !ok {
			return nil, errNoUDPEndpoint
		}
		var target v4wire.Pubkey
		crand.Read(target[:])
		peers, err := t.findnode(dst.ID(), addr, target)
		if err != nil {
			t.log.Trace("FINDNODE failed", "id", dst.ID(), "err", err)
		}
		return peers, err
	}
	return newCrawlIterator(opts, queryFn)
}

// CrawlIterator returns an enode.Iterator that performs a breadth-first
// crawl using single-distance FINDNODE requests. See [UDPv4.CrawlIterator]
// for the algorithm; the discv5 protocol takes a list of distances directly,
// so the rotation maps to distances [256, 255, ..., 256-Drange+1].
func (t *UDPv5) CrawlIterator(opts CrawlOptions) enode.Iterator {
	queryFn := func(dst *enode.Node, d int) ([]*enode.Node, error) {
		dist := uint(256 - d)
		peers, err := t.Findnode(dst, []uint{dist})
		if err != nil {
			t.log.Trace("FINDNODE failed", "id", dst.ID(), "err", err)
		}
		return peers, err
	}
	return newCrawlIterator(opts, queryFn)
}

// crawlIterator is a breadth-first FINDNODE-driven iterator. It maintains a
// shared work queue and an output buffer; workers pop from the queue, issue
// one FINDNODE per pop, and feed any newly-seen peers back into both the
// queue and the output buffer. The iterator terminates when the queue is
// empty and no FINDNODE call is in flight.
type crawlIterator struct {
	queryFn   func(dst *enode.Node, d int) ([]*enode.Node, error)
	drange    int
	outputCap int
	wg        sync.WaitGroup

	mu         sync.Mutex
	cond       *sync.Cond
	queue      []*enode.Node // pending FINDNODE work
	output     []*enode.Node // emitted peers (one-time)
	discovered map[enode.ID]struct{}
	inflight   int  // queued + in-progress
	closing    bool // Close() called or natural termination
	cur        *enode.Node

	rotation atomic.Uint64
}

func newCrawlIterator(opts CrawlOptions, queryFn func(*enode.Node, int) ([]*enode.Node, error)) *crawlIterator {
	opts.withDefaults()
	it := &crawlIterator{
		queryFn:    queryFn,
		drange:     opts.Drange,
		outputCap:  opts.OutputCap,
		discovered: make(map[enode.ID]struct{}),
	}
	it.cond = sync.NewCond(&it.mu)

	// Seed directly into the queue/output. Going through discover() would
	// block on the OutputCap if len(Seeds) > OutputCap, deadlocking the
	// constructor since workers haven't started and Next() hasn't been
	// called yet.
	for _, n := range opts.Seeds {
		if n == nil {
			continue
		}
		if _, seen := it.discovered[n.ID()]; seen {
			continue
		}
		it.discovered[n.ID()] = struct{}{}
		it.queue = append(it.queue, n)
		it.output = append(it.output, n)
		it.inflight++
	}

	// Workers. The first (Workers - RandomWorkers) workers pop FIFO; the
	// remaining RandomWorkers pop a uniform-random queue index. Both share
	// the same queue, mutex, cond, and termination semantics.
	fifoCount := opts.Workers - opts.RandomWorkers
	for i := 0; i < fifoCount; i++ {
		it.wg.Add(1)
		go it.worker(false)
	}
	for i := 0; i < opts.RandomWorkers; i++ {
		it.wg.Add(1)
		go it.worker(true)
	}
	return it
}

// discover records a newly-seen peer. Acquires mu internally; callers
// MUST NOT hold it. If output is at capacity, waits on cond until Next()
// drains it; this is the iterator's backpressure point.
func (it *crawlIterator) discover(n *enode.Node) {
	if n == nil {
		return
	}
	it.mu.Lock()
	defer it.mu.Unlock()
	for {
		if it.closing {
			return
		}
		if _, seen := it.discovered[n.ID()]; seen {
			return
		}
		if it.outputCap > 0 && len(it.output) >= it.outputCap {
			// Pause discovery until the consumer drains output. Releases mu
			// while waiting so other workers can keep popping from queue and
			// the consumer can pop from output.
			it.cond.Wait()
			continue
		}
		break
	}
	it.discovered[n.ID()] = struct{}{}
	it.queue = append(it.queue, n)
	it.output = append(it.output, n)
	it.inflight++
	it.cond.Broadcast()
}

// popWork blocks until either a peer is available to query, or the iterator
// has nothing left to do. Returns (nil, false) on termination. If random is
// true, pops a uniform-random index via swap-and-pop; otherwise pops the
// FIFO front. Both modes share the same termination semantics.
func (it *crawlIterator) popWork(random bool) (*enode.Node, bool) {
	it.mu.Lock()
	defer it.mu.Unlock()
	for {
		if it.closing {
			return nil, false
		}
		if len(it.queue) > 0 {
			if random {
				i := rand.IntN(len(it.queue))
				n := it.queue[i]
				// Swap-and-pop: O(1) removal from middle.
				it.queue[i] = it.queue[len(it.queue)-1]
				it.queue = it.queue[:len(it.queue)-1]
				return n, true
			}
			n := it.queue[0]
			it.queue = it.queue[1:]
			return n, true
		}
		if it.inflight == 0 {
			// Queue empty AND nothing in flight: natural termination.
			it.closing = true
			it.cond.Broadcast()
			return nil, false
		}
		it.cond.Wait()
	}
}

// finishWork is called by workers after their FINDNODE response has been
// processed. It decrements the in-flight counter and broadcasts so a possibly
// idle worker can re-evaluate termination.
func (it *crawlIterator) finishWork() {
	it.mu.Lock()
	defer it.mu.Unlock()
	it.inflight--
	if it.inflight == 0 && len(it.queue) == 0 {
		it.closing = true
		it.cond.Broadcast()
	}
}

func (it *crawlIterator) worker(random bool) {
	defer it.wg.Done()
	for {
		n, ok := it.popWork(random)
		if !ok {
			return
		}
		d := int(it.rotation.Add(1)-1) % it.drange
		peers, _ := it.queryFn(n, d)
		for _, p := range peers {
			it.discover(p)
		}
		it.finishWork()
	}
}

// Next blocks until a newly-discovered peer is available, then returns true
// and makes the peer accessible via Node. Returns false when the iterator
// has terminated.
func (it *crawlIterator) Next() bool {
	it.mu.Lock()
	defer it.mu.Unlock()
	for len(it.output) == 0 {
		if it.closing {
			return false
		}
		it.cond.Wait()
	}
	it.cur = it.output[0]
	it.output = it.output[1:]
	// Wake any worker stalled in discover() because output was at capacity.
	it.cond.Broadcast()
	return true
}

// Node returns the most recent peer surfaced by Next.
func (it *crawlIterator) Node() *enode.Node {
	it.mu.Lock()
	defer it.mu.Unlock()
	return it.cur
}

// Close terminates the iterator, unblocking any goroutines waiting in Next.
// Workers exit at their next poll point; in-flight FINDNODE responses are
// dropped.
func (it *crawlIterator) Close() {
	it.mu.Lock()
	if !it.closing {
		it.closing = true
		it.cond.Broadcast()
	}
	it.mu.Unlock()
	it.wg.Wait()
}
