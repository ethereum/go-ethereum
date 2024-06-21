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
	if n := tr.fast.get(now, &tab.rand, tr.activeReq); n != nil {
		tr.startRequest(tab, n)
		tr.fast.schedule(now, &tab.rand)
	}
	if n := tr.slow.get(now, &tab.rand, tr.activeReq); n != nil {
		tr.startRequest(tab, n)
		tr.slow.schedule(now, &tab.rand)
	}

	return min(tr.fast.nextTime, tr.slow.nextTime)
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
}

// get returns a random node from the queue. Nodes in the 'exclude' map are not returned.
func (list *revalidationList) get(now mclock.AbsTime, rand randomSource, exclude map[enode.ID]struct{}) *tableNode {
	if now < list.nextTime || len(list.nodes) == 0 {
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

func (list *revalidationList) schedule(now mclock.AbsTime, rand randomSource) {
	list.nextTime = now.Add(time.Duration(rand.Int63n(int64(list.interval))))
}

func (list *revalidationList) push(n *tableNode, now mclock.AbsTime, rand randomSource) {
	list.nodes = append(list.nodes, n)
	if list.nextTime == never {
		list.schedule(now, rand)
	}
	n.revalList = list
}

func (list *revalidationList) remove(n *tableNode) {
	i := slices.Index(list.nodes, n)
	if i == -1 {
		panic(fmt.Errorf("node %v not found in list", n.ID()))
	}
	list.nodes = slices.Delete(list.nodes, i, i+1)
	if len(list.nodes) == 0 {
		list.nextTime = never
	}
	n.revalList = nil
}

func (list *revalidationList) contains(id enode.ID) bool {
	return slices.ContainsFunc(list.nodes, func(n *tableNode) bool {
		return n.ID() == id
	})
}
