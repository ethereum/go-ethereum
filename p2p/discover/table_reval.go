// Copyright 2024 The go-ethereum Authors
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
	"fmt"
	"math"
	"slices"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

const never = mclock.AbsTime(math.MaxInt64)

const slowRevalidationFactor = 3

// tableRevalidation implements the node revalidation process.
// It tracks all nodes contained in Table, and schedules sending PING to them.
type tableRevalidation struct {
	fast      revalidationList
	slow      revalidationList
	activeReq map[enode.ID]struct{}
}

type revalidationResponse struct {
	n          *tableNode
	newRecord  *enode.Node
	didRespond bool
}

func (tr *tableRevalidation) init(cfg *Config) {
	tr.activeReq = make(map[enode.ID]struct{})
	tr.fast.nextTime = never
	tr.fast.interval = cfg.PingInterval
	tr.fast.name = "fast"
	tr.slow.nextTime = never
	tr.slow.interval = cfg.PingInterval * slowRevalidationFactor
	tr.slow.name = "slow"
}

// nodeAdded is called when the table receives a new node.
func (tr *tableRevalidation) nodeAdded(tab *Table, n *tableNode) {
	tr.fast.push(n, tab.cfg.Clock.Now(), &tab.rand)
}

// nodeRemoved is called when a node was removed from the table.
func (tr *tableRevalidation) nodeRemoved(n *tableNode) {
	if n.revalList == nil {
		panic(fmt.Errorf("removed node %v has nil revalList", n.ID()))
	}
	n.revalList.remove(n)
}

// nodeEndpointChanged is called when a change in IP or port is detected.
func (tr *tableRevalidation) nodeEndpointChanged(tab *Table, n *tableNode) {
	n.isValidatedLive = false
	tr.moveToList(&tr.fast, n, tab.cfg.Clock.Now(), &tab.rand)
}

// run performs node revalidation.
// It returns the next time it should be invoked, which is used in the Table main loop
// to schedule a timer. However, run can be called at any time.
func (tr *tableRevalidation) run(tab *Table, now mclock.AbsTime) (nextTime mclock.AbsTime) {
	reval := func(list *revalidationList) {
		list.mu.Lock()
		shouldSchedule := list.nextTime <= now
		list.mu.Unlock()

		if shouldSchedule {
			if n := list.get(&tab.rand, tr.activeReq); n != nil {
				tr.startRequest(tab, n)
			}
			// Update nextTime regardless if any requests were started because
			// current value has passed.
			list.schedule(now, &tab.rand)
		}
	}
	reval(&tr.fast)
	reval(&tr.slow)

	tr.fast.mu.Lock()
	fastNext := tr.fast.nextTime
	tr.fast.mu.Unlock()
	tr.slow.mu.Lock()
	slowNext := tr.slow.nextTime
	tr.slow.mu.Unlock()
	return min(fastNext, slowNext)
}

// startRequest spawns a revalidation request for node n.
func (tr *tableRevalidation) startRequest(tab *Table, n *tableNode) {
	if _, ok := tr.activeReq[n.ID()]; ok {
		panic(fmt.Errorf("duplicate startRequest (node %v)", n.ID()))
	}
	tr.activeReq[n.ID()] = struct{}{}
	resp := revalidationResponse{n: n}

	// Fetch the node while holding lock.
	tab.mutex.Lock()
	node := n.Node
	tab.mutex.Unlock()

	go tab.doRevalidate(resp, node)
}

func (tab *Table) doRevalidate(resp revalidationResponse, node *enode.Node) {
	// Ping the selected node and wait for a pong response.
	remoteSeq, err := tab.net.ping(node)
	resp.didRespond = err == nil

	// Also fetch record if the node replied and returned a higher sequence number.
	if remoteSeq > node.Seq() {
		newrec, err := tab.net.RequestENR(node)
		if err != nil {
			tab.log.Debug("ENR request failed", "id", node.ID(), "err", err)
		} else {
			resp.newRecord = newrec
		}
	}

	select {
	case tab.revalResponseCh <- resp:
	case <-tab.closed:
	}
}

// handleResponse processes the result of a revalidation request.
func (tr *tableRevalidation) handleResponse(tab *Table, resp revalidationResponse) {
	var (
		now = tab.cfg.Clock.Now()
		n   = resp.n
		b   = tab.bucket(n.ID())
	)
	delete(tr.activeReq, n.ID())

	// If the node was removed from the table while getting checked, we need to stop
	// processing here to avoid re-adding it.
	if n.revalList == nil {
		return
	}

	// Store potential seeds in database.
	// This is done via defer to avoid holding Table lock while writing to DB.
	defer func() {
		if n.isValidatedLive && n.livenessChecks > 5 {
			tab.db.UpdateNode(resp.n.Node)
		}
	}()

	// Remaining logic needs access to Table internals.
	tab.mutex.Lock()
	defer tab.mutex.Unlock()

	if !resp.didRespond {
		n.livenessChecks /= 3
		if n.livenessChecks <= 0 {
			tab.deleteInBucket(b, n.ID())
		} else {
			tab.log.Debug("Node revalidation failed", "b", b.index, "id", n.ID(), "checks", n.livenessChecks, "q", n.revalList.name)
			tr.moveToList(&tr.fast, n, now, &tab.rand)
		}
		return
	}

	// The node responded.
	n.livenessChecks++
	n.isValidatedLive = true
	tab.log.Debug("Node revalidated", "b", b.index, "id", n.ID(), "checks", n.livenessChecks, "q", n.revalList.name)
	var endpointChanged bool
	if resp.newRecord != nil {
		_, endpointChanged = tab.bumpInBucket(b, resp.newRecord, false)
	}

	// Node moves to slow list if it passed and hasn't changed.
	if !endpointChanged {
		tr.moveToList(&tr.slow, n, now, &tab.rand)
	}
}

// moveToList ensures n is in the 'dest' list.
func (tr *tableRevalidation) moveToList(dest *revalidationList, n *tableNode, now mclock.AbsTime, rand randomSource) {
	if n.revalList == dest {
		return
	}
	if n.revalList != nil {
		n.revalList.remove(n)
	}
	dest.push(n, now, rand)
}

// revalidationList holds a list nodes and the next revalidation time.
type revalidationList struct {
	nodes    []*tableNode
	nextTime mclock.AbsTime
	interval time.Duration
	name     string
	mu       sync.Mutex
}

// get returns a random node from the queue. Nodes in the 'exclude' map are not returned.
func (list *revalidationList) get(rand randomSource, exclude map[enode.ID]struct{}) *tableNode {
	if len(list.nodes) == 0 {
		return nil
	}
	for i := 0; i < len(list.nodes)*3; i++ {
		n := list.nodes[rand.Intn(len(list.nodes))]
		_, excluded := exclude[n.ID()]
		if !excluded {
			return n
		}
	}
	return nil
}

// schedule computes the next revalidation time.
func (list *revalidationList) schedule(now mclock.AbsTime, rand randomSource) {
	list.mu.Lock()         // Lock before accessing nextTime
	defer list.mu.Unlock() // Unlock when function exits

	if len(list.nodes) == 0 {
		list.nextTime = never
		return
	}
	// Add random delay up to the interval duration.
	// This ensures nodes are revalidated close to the interval on average,
	// but not all at the same time.
	delay := time.Duration(rand.Int63n(int64(list.interval)))
	list.nextTime = now.Add(delay)
}

func (list *revalidationList) push(n *tableNode, now mclock.AbsTime, rand randomSource) {
	list.mu.Lock()
	list.nodes = append(list.nodes, n)
	n.revalList = list
	list.mu.Unlock()

	// If list was previously empty, reschedule. schedule handles its own locking.
	if len(list.nodes) == 1 {
		list.schedule(now, rand)
	}
}

func (list *revalidationList) remove(n *tableNode) {
	list.mu.Lock()
	defer list.mu.Unlock()

	if n.revalList != list {
		panic(fmt.Errorf("node %v is not in list %q", n.ID(), list.name))
	}
	idx := -1
	for i, node := range list.nodes {
		if node == n {
			idx = i
			break
		}
	}
	if idx == -1 {
		panic(fmt.Errorf("node %v is not in list %q (but revalList field points to it)", n.ID(), list.name))
	}
	list.nodes = slices.Delete(list.nodes, idx, idx+1)
	n.revalList = nil // Mark node as removed from any list
}

func (list *revalidationList) contains(id enode.ID) bool {
	return slices.ContainsFunc(list.nodes, func(n *tableNode) bool {
		return n.ID() == id
	})
}
