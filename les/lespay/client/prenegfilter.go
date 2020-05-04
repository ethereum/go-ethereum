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

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nodestate"
)

// PreNegFilter is a filter on an enode.Iterator that performs connection pre-negotiation
// using the provided callback and only returns nodes that gave a positive answer recently.
type PreNegFilter struct {
	lock                            sync.Mutex
	cond                            *sync.Cond
	ns                              *nodestate.NodeStateMachine
	sfQueried, sfCanDial            nodestate.Flags
	queryTimeout, canDialTimeout    time.Duration
	input, canDialIter              enode.Iterator
	query                           PreNegQuery
	pending                         map[*enode.Node]func()
	pendingQueries, needQueries     int
	maxPendingQueries, canDialCount int
	waitingForNext, closed          bool
	testClock                       *mclock.Simulated
}

// PreNegQuery callback performs connection pre-negotiation.
// Note: the result callback function should always be called, if it has not been called
// before then cancel should call it.
type PreNegQuery func(n *enode.Node, result func(canDial bool)) (start, cancel func())

// NewPreNegFilter creates a new PreNegFilter. sfQueried is set for each queried node, sfCanDial
// is set together with sfQueried being reset if the callback returned a positive answer. The output
// iterator returns nodes with an active sfCanDial flag but does not automatically reset the flag
// (the dialer can do that together with setting the dialed flag).
// The filter starts at most the specified number of simultaneous queries if there are no nodes
// with an active sfCanDial flag and the output iterator is already being read. Note that until
// sfCanDial is reset or times out the filter won't start more queries even if the dial candidate
// has been returned by the output iterator.
// If a simulated clock is used for testing then it should be provided in order to advance
// clock when waiting for query results.
func NewPreNegFilter(ns *nodestate.NodeStateMachine, input enode.Iterator, query PreNegQuery, sfQueried, sfCanDial nodestate.Flags, maxPendingQueries int, queryTimeout, canDialTimeout time.Duration, testClock *mclock.Simulated) *PreNegFilter {
	pf := &PreNegFilter{
		ns:                ns,
		input:             input,
		query:             query,
		sfQueried:         sfQueried,
		sfCanDial:         sfCanDial,
		queryTimeout:      queryTimeout,
		canDialTimeout:    canDialTimeout,
		maxPendingQueries: maxPendingQueries,
		canDialIter:       NewQueueIterator(ns, sfCanDial, nodestate.Flags{}, false),
		pending:           make(map[*enode.Node]func()),
		testClock:         testClock,
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
			// query timeout, call cancel function (the query result callback will remove it from the map)
			cancel = pf.pending[n]
		}
		pf.lock.Unlock()
		if cancel != nil {
			cancel()
		}
	})
	go pf.readLoop()
	return pf
}

// checkQuery checks whether we need more queries and signals readLoop if necessary
func (pf *PreNegFilter) checkQuery() {
	if pf.waitingForNext && pf.canDialCount == 0 {
		pf.needQueries = pf.maxPendingQueries
	}
	if pf.needQueries > pf.pendingQueries {
		pf.cond.Signal()
	}
}

// readLoop reads nodes from the input iterator and starts new queries if necessary
func (pf *PreNegFilter) readLoop() {
	for {
		pf.lock.Lock()
		if pf.testClock != nil {
			for pf.pendingQueries == pf.maxPendingQueries {
				// advance simulated clock until our queries are finished or timed out
				pf.lock.Unlock()
				pf.testClock.Run(time.Second)
				pf.lock.Lock()
				if pf.closed {
					pf.lock.Unlock()
					return
				}
			}
		}
		for pf.needQueries <= pf.pendingQueries {
			// either no queries are needed or we have enough pending; wait until more
			// are needed
			pf.cond.Wait()
			if pf.closed {
				pf.lock.Unlock()
				return
			}
		}
		pf.lock.Unlock()
		// fetch a node from the input that is not pending at the moment
		var node *enode.Node
		for {
			if !pf.input.Next() {
				pf.canDialIter.Close()
				return
			}
			node = pf.input.Node()
			pf.lock.Lock()
			_, pending := pf.pending[node]
			pf.lock.Unlock()
			if !pending {
				break
			}
		}
		// set sfQueried and start the query
		pf.ns.SetState(node, pf.sfQueried, nodestate.Flags{}, pf.queryTimeout)
		start, cancel := pf.query(node, func(canDial bool) {
			if canDial {
				pf.lock.Lock()
				pf.needQueries = 0
				pf.pendingQueries--
				delete(pf.pending, node)
				pf.lock.Unlock()
				pf.ns.SetState(node, pf.sfCanDial, pf.sfQueried, pf.canDialTimeout)
			} else {
				pf.ns.SetState(node, nodestate.Flags{}, pf.sfQueried, 0)
				pf.lock.Lock()
				pf.pendingQueries--
				delete(pf.pending, node)
				pf.checkQuery()
				pf.lock.Unlock()
			}
		})
		// add pending entry before actually starting
		pf.lock.Lock()
		pf.pendingQueries++
		pf.pending[node] = cancel
		pf.lock.Unlock()
		start()
	}
}

// Next moves to the next selectable node.
func (pf *PreNegFilter) Next() bool {
	pf.lock.Lock()
	pf.waitingForNext = true
	// start queries if we cannot give a result immediately
	pf.checkQuery()
	pf.lock.Unlock()
	// get a result from the LIFO queue that returns nodes with active sfCanDial
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
