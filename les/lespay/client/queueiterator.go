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
)

// QueueIterator returns nodes from the specified selectable set in the same order as
// they entered the set.
type QueueIterator struct {
	lock     sync.Mutex
	ns       *utils.NodeStateMachine
	queue    []*enode.Node
	selected utils.NodeStateBitMask
	wakeup   chan struct{}
	nextNode *enode.Node
	closed   bool
}

// NewQueueIterator creates a new QueueIterator. Nodes are selectable if they have all the required
// and none of the disabled flags set. When a node is selected the selectedFlag is set which also
// disables further selectability until it is removed or times out.
// The ENR field should be set for all selectable nodes so that the iterator can return complete enodes.
func NewQueueIterator(ns *utils.NodeStateMachine, requireMask, disableMask utils.NodeStateBitMask, selectedFlag *utils.NodeStateFlag) *QueueIterator {

	selected := ns.StateMask(selectedFlag)
	disableMask |= selected
	qi := &QueueIterator{
		ns:       ns,
		selected: selected,
	}
	ns.SubscribeState(requireMask|disableMask, func(n *enode.Node, oldState, newState utils.NodeStateBitMask) {
		oldMatch := (oldState&requireMask == requireMask) && (oldState&disableMask == 0)
		newMatch := (newState&requireMask == requireMask) && (newState&disableMask == 0)
		if newMatch != oldMatch {
			qi.lock.Lock()
			if newMatch {
				qi.queue = append(qi.queue, n)
				if qi.wakeup != nil {
					close(qi.wakeup)
					qi.wakeup = nil
				}
			} else {
				id := n.ID()
				for i, qn := range qi.queue {
					if qn.ID() == id {
						copy(qi.queue[i:len(qi.queue)-1], qi.queue[i+1:])
						qi.queue = qi.queue[:len(qi.queue)-1]
						break
					}
				}
			}
			qi.lock.Unlock()
		}
	})
	return qi
}

// Next implements enode.Iterator
func (qi *QueueIterator) Next() bool {
	qi.lock.Lock()
	for {
		if qi.closed {
			qi.lock.Unlock()
			return false
		}
		if len(qi.queue) > 0 {
			qi.nextNode = qi.queue[0]
			copy(qi.queue[:len(qi.queue)-1], qi.queue[1:])
			qi.queue = qi.queue[:len(qi.queue)-1]
			qi.lock.Unlock()
			qi.ns.SetState(qi.nextNode, qi.selected, 0, time.Second*5)
			return true
		}
		ch := make(chan struct{})
		qi.wakeup = ch
		qi.lock.Unlock()
		<-ch
		qi.lock.Lock()
	}
}

// Close implements enode.Iterator
func (qi *QueueIterator) Close() {
	qi.lock.Lock()
	defer qi.lock.Unlock()

	qi.closed = true
	if qi.wakeup != nil {
		close(qi.wakeup)
		qi.wakeup = nil
	}
}

// Node implements enode.Iterator
func (qi *QueueIterator) Node() *enode.Node {
	qi.lock.Lock()
	defer qi.lock.Unlock()

	return qi.nextNode
}
