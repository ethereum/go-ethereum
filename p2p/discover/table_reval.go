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
	"math"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

const never = mclock.AbsTime(math.MaxInt64)

type tableRevalidation struct {
	newNodes  revalidationList
	nodes     revalidationList
	activeReq map[enode.ID]struct{}
}

type revalidationResponse struct {
	n          *node
	didRespond bool
	isNewNode  bool
	newRecord  *enode.Node
}

func (tr *tableRevalidation) init(cfg *Config) {
	tr.activeReq = make(map[enode.ID]struct{})
	tr.newNodes.nextTime = never
	tr.newNodes.interval = cfg.PingInterval / 3
	tr.nodes.nextTime = never
	tr.nodes.interval = cfg.PingInterval
}

// nodeAdded is called when the table receives a new node.
func (tr *tableRevalidation) nodeAdded(tab *Table, n *node) {
	tr.newNodes.push(n, tab.cfg.Clock.Now(), tab.rand)
}

// nodeRemoved is called when a node was removed from the table.
func (tr *tableRevalidation) nodeRemoved(n *node) {
	wasnew := tr.newNodes.remove(n)
	if !wasnew {
		tr.nodes.remove(n)
	}
}

// nextTime returns the next time run() should be invoked.
// The Table main loop uses this to schedule a timer.
func (tr *tableRevalidation) nextTime() mclock.AbsTime {
	return min(tr.newNodes.nextTime, tr.nodes.nextTime)
}

// run performs node revalidation.
func (tr *tableRevalidation) run(tab *Table, now mclock.AbsTime) {
	if n := tr.newNodes.get(now, tab.rand, tr.activeReq); n != nil {
		tr.startRequest(tab, n, true)
		tr.newNodes.schedule(now, tab.rand)
	}
	if n := tr.nodes.get(now, tab.rand, tr.activeReq); n != nil {
		tr.startRequest(tab, n, false)
		tr.nodes.schedule(now, tab.rand)
	}
}

// startRequest spawns a revalidation request for node n.
func (tr *tableRevalidation) startRequest(tab *Table, n *node, newNode bool) {
	if _, ok := tr.activeReq[n.ID()]; ok {
		panic("duplicate startRequest")
	}
	tr.activeReq[n.ID()] = struct{}{}
	resp := revalidationResponse{n: n, isNewNode: newNode}

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
	case tab.revalidateResp <- resp:
	case <-tab.closed:
	}
}

// handleResponse processes the result of a revalidation request.
func (tr *tableRevalidation) handleResponse(tab *Table, resp revalidationResponse) {
	n := resp.n
	b := tab.bucket(n.ID())
	delete(tr.activeReq, n.ID())

	tab.mutex.Lock()
	defer tab.mutex.Unlock()

	if !resp.didRespond {
		// Revalidation failed.
		n.livenessChecks /= 3
		if n.livenessChecks == 0 || resp.isNewNode {
			tab.deleteInBucket(b, n.ID())
		}
		return
	}

	// The node responded.
	n.livenessChecks++
	n.isValidatedLive = true
	tab.log.Debug("Revalidated node", "b", b.index, "id", n.ID(), "checks", n.livenessChecks)
	if resp.newRecord != nil {
		updated := tab.bumpInBucket(b, resp.newRecord)
		if updated {
			// If the node changed its advertised endpoint, the updated ENR is not served
			// until it has been revalidated.
			n.isValidatedLive = false
		}
	}

	// Move node over to main queue after first validation.
	if resp.isNewNode {
		tr.newNodes.remove(n)
		tr.nodes.push(n, tab.cfg.Clock.Now(), tab.rand)
	}

	// Store potential seeds in database.
	if n.isValidatedLive && n.livenessChecks > 5 {
		tab.db.UpdateNode(resp.n.Node)
	}
}

// revalidationList holds a list nodes and the next revalidation time.
type revalidationList struct {
	nodes    []*node
	nextTime mclock.AbsTime
	interval time.Duration
}

// get returns a random node from the queue. Nodes in the 'exclude' map are not returned.
func (rq *revalidationList) get(now mclock.AbsTime, rand randomSource, exclude map[enode.ID]struct{}) *node {
	if now < rq.nextTime || len(rq.nodes) == 0 {
		return nil
	}
	for i := 0; i < len(rq.nodes)*3; i++ {
		n := rq.nodes[rand.Intn(len(rq.nodes))]
		_, excluded := exclude[n.ID()]
		if !excluded {
			return n
		}
	}
	return nil
}

func (rq *revalidationList) push(n *node, now mclock.AbsTime, rand randomSource) {
	rq.nodes = append(rq.nodes, n)
	if rq.nextTime == never {
		rq.schedule(now, rand)
	}
}

func (rq *revalidationList) schedule(now mclock.AbsTime, rand randomSource) {
	rq.nextTime = now.Add(time.Duration(rand.Int63n(int64(rq.interval))))
}

func (rq *revalidationList) remove(n *node) bool {
	i := slices.Index(rq.nodes, n)
	if i == -1 {
		return false
	}
	rq.nodes = slices.Delete(rq.nodes, i, i+1)
	if len(rq.nodes) == 0 {
		rq.nextTime = never
	}
	return true
}
