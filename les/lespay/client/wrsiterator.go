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

	"github.com/ethereum/go-ethereum/les/utils"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nodestate"
)

// WrsIterator returns nodes from the specified selectable set with a weighted random
// selection. Selection weights are provided by a callback function.
type WrsIterator struct {
	lock     sync.Mutex
	ns       *nodestate.NodeStateMachine
	wrs      *utils.WeightedRandomSelect
	selected nodestate.Flags
	wakeup   chan struct{}
	nextNode *enode.Node
	closed   bool
}

// NewWrsIterator creates a new WrsIterator. Nodes are selectable if they have all the required
// and none of the disabled flags set. When a node is selected the selectedFlag is set which also
// disables further selectability until it is removed or times out.
// The ENR field should be set for all selectable nodes so that the iterator can return complete enodes.
func NewWrsIterator(ns *nodestate.NodeStateMachine, requireFlags, disableFlags, selectedFlag nodestate.Flags, wfn func(interface{}) uint64) *WrsIterator {
	disableFlags = disableFlags.Or(selectedFlag)
	w := &WrsIterator{
		ns:       ns,
		wrs:      utils.NewWeightedRandomSelect(wfn),
		selected: selectedFlag,
	}
	ns.SubscribeState(requireFlags.Or(disableFlags), func(n *enode.Node, oldState, newState nodestate.Flags) {
		oldMatch := oldState.HasAll(requireFlags) && oldState.HasNone(disableFlags)
		newMatch := newState.HasAll(requireFlags) && newState.HasNone(disableFlags)
		if newMatch == oldMatch {
			return
		}

		w.lock.Lock()
		if newMatch {
			w.wrs.Update(n.ID())
			if w.wakeup != nil && !w.wrs.IsEmpty() {
				close(w.wakeup)
				w.wakeup = nil
			}
		} else {
			w.wrs.Remove(n.ID())
		}
		w.lock.Unlock()
	})
	return w
}

// Next implements enode.Iterator
func (w *WrsIterator) Next() bool {
	w.lock.Lock()
	for {
		if w.closed {
			w.lock.Unlock()
			return false
		}
		var node *enode.Node
		if c := w.wrs.Choose(); c != nil {
			node = w.ns.GetNode(c.(enode.ID))
		}
		if node != nil {
			w.nextNode = node
			w.lock.Unlock()
			w.ns.SetState(w.nextNode, w.selected, nodestate.Flags{}, time.Second*5)
			return true
		}
		ch := make(chan struct{})
		w.wakeup = ch
		w.lock.Unlock()
		<-ch
		w.lock.Lock()
	}
}

// Close implements enode.Iterator
func (w *WrsIterator) Close() {
	w.lock.Lock()
	defer w.lock.Unlock()

	w.closed = true
	if w.wakeup != nil {
		close(w.wakeup)
		w.wakeup = nil
	}
}

// Node implements enode.Iterator
func (w *WrsIterator) Node() *enode.Node {
	w.lock.Lock()
	defer w.lock.Unlock()

	return w.nextNode
}
