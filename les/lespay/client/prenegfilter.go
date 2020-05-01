// Copyright 2020 The go-ethereum Authors
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

package client

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nodestate"
)

type PreNegFilter struct {
	lock                            sync.Mutex
	cond                            *sync.Cond
	ns                              *nodestate.NodeStateMachine
	sfQueried, sfCanDial            nodestate.Flags
	input, canDialIter              enode.Iterator
	query                           PreNegQuery
	cancel                          map[*enode.Node]func()
	pendingQueries, needQueries     int
	maxPendingQueries, canDialCount int
	waitingForNext, closed          bool
}

// The result callback function should always be called, even if the query is cancelled.
type PreNegQuery func(n *enode.Node, result func(canDial bool)) (start, cancel func())

func NewPreNegFilter(ns *nodestate.NodeStateMachine, input enode.Iterator, query PreNegQuery, sfQueried, sfCanDial nodestate.Flags, maxPendingQueries int) *PreNegFilter {
	pf := &PreNegFilter{
		ns:                ns,
		input:             input,
		query:             query,
		sfQueried:         sfQueried,
		sfCanDial:         sfCanDial,
		maxPendingQueries: maxPendingQueries,
		canDialIter:       NewQueueIterator(ns, sfCanDial, nodestate.Flags{}, false),
		cancel:            make(map[*enode.Node]func()),
	}
	pf.cond = sync.NewCond(&pf.lock)
	ns.SubscribeState(sfQueried.Or(sfCanDial), func(n *enode.Node, oldState, newState nodestate.Flags) {
		var cancel func()
		pf.lock.Lock()
		if oldState.HasAll(sfCanDial) {
			pf.canDialCount--
		}
		if newState.HasAll(sfCanDial) {
			pf.canDialCount++
		}
		pf.checkQuery()
		if oldState.HasAll(sfQueried) && newState.HasNone(sfQueried.Or(sfCanDial)) {
			// query timeout, call cancel function
			cancel = pf.cancel[n]
		}
		pf.lock.Unlock()
		if cancel != nil {
			cancel()
		}
	})
	go pf.readLoop()
	return pf
}

func (pf *PreNegFilter) checkQuery() {
	if pf.waitingForNext && pf.canDialCount == 0 {
		pf.needQueries = pf.maxPendingQueries
	}
	if pf.needQueries > pf.pendingQueries {
		pf.cond.Signal()
	}
}

func (pf *PreNegFilter) readLoop() {
	for {
		pf.lock.Lock()
		for pf.needQueries <= pf.pendingQueries {
			pf.cond.Wait()
			if pf.closed {
				pf.lock.Unlock()
				return
			}
		}
		pf.lock.Unlock()
		if !pf.input.Next() {
			return
		}
		node := pf.input.Node()
		pf.ns.SetState(node, pf.sfQueried, nodestate.Flags{}, time.Second*5)
		start, cancel := pf.query(node, func(canDial bool) {
			if canDial {
				pf.lock.Lock()
				pf.needQueries = 0
				pf.pendingQueries--
				delete(pf.cancel, node)
				pf.lock.Unlock()
				pf.ns.SetState(node, pf.sfCanDial, pf.sfQueried, time.Second*10)
			} else {
				pf.ns.SetState(node, nodestate.Flags{}, pf.sfQueried, 0)
				pf.lock.Lock()
				pf.pendingQueries--
				delete(pf.cancel, node)
				pf.checkQuery()
				pf.lock.Unlock()
			}
		})
		pf.lock.Lock()
		pf.pendingQueries++
		pf.cancel[node] = cancel
		pf.lock.Unlock()
		start()
	}
}

// Next moves to the next selectable node.
func (pf *PreNegFilter) Next() bool {
	pf.lock.Lock()
	pf.waitingForNext = true
	pf.checkQuery()
	pf.lock.Unlock()
	next := pf.canDialIter.Next()
	pf.lock.Lock()
	pf.needQueries = 0
	pf.waitingForNext = false
	pf.lock.Unlock()
	return next
}

// Close ends the iterator.
func (pf *PreNegFilter) Close() {
	pf.lock.Lock()
	pf.closed = true
	pf.cond.Signal()
	pf.lock.Unlock()
	pf.input.Close()
	pf.canDialIter.Close()
}

// Node returns the current node.
func (pf *PreNegFilter) Node() *enode.Node {
	return pf.canDialIter.Node()
}
