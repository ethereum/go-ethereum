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
	"github.com/ethereum/go-ethereum/p2p/enr"
)

// WrsIterator returns nodes from the specified selectable set with a weighted random
// selection. Selection weights are provided by a callback function.
type WrsIterator struct {
	lock                               sync.Mutex
	ns                                 *utils.NodeStateMachine
	wrs                                *utils.WeightedRandomSelect
	requireMask, disableMask, selected utils.NodeStateBitMask
	enrFieldID                         int
	validSchemes                       enr.IdentityScheme
	wakeup                             chan struct{}
	nextID                             enode.ID
	nextENR                            *enr.Record
	closed                             bool
}

// NewWrsIterator creates a new WrsIterator. Nodes are selectable if they have all the required
// and none of the disabled flags set. When a node is selected the selectedFlag is set which also
// disables further selectability until it is removed or times out.
// The ENR field should be set for all selectable nodes so that the iterator can return complete enodes.
func NewWrsIterator(ns *utils.NodeStateMachine, requireMask, disableMask utils.NodeStateBitMask, selectedFlag *utils.NodeStateFlag,
	enrField *utils.NodeField, wfn func(interface{}) uint64, validSchemes enr.IdentityScheme) *WrsIterator {

	selected := ns.StateMask(selectedFlag)
	disableMask |= selected
	w := &WrsIterator{
		ns:           ns,
		wrs:          utils.NewWeightedRandomSelect(wfn),
		requireMask:  requireMask,
		disableMask:  disableMask,
		selected:     selected,
		enrFieldID:   ns.FieldIndex(enrField),
		validSchemes: validSchemes,
	}
	ns.SubscribeState(requireMask|disableMask, func(id enode.ID, oldState, newState utils.NodeStateBitMask) {
		ps := (oldState&w.disableMask) == 0 && (oldState&w.requireMask) == w.requireMask
		ns := (newState&w.disableMask) == 0 && (newState&w.requireMask) == w.requireMask

		w.lock.Lock()
		defer w.lock.Unlock()

		if ps == ns {
			return
		}
		if ns {
			w.wrs.Update(id)
			if w.wakeup != nil && !w.wrs.IsEmpty() {
				close(w.wakeup)
				w.wakeup = nil
			}
		} else {
			w.wrs.Remove(id)
		}
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
		n := w.wrs.Choose()
		if n != nil {
			w.nextID = n.(enode.ID)
			e := w.ns.GetField(w.nextID, w.enrFieldID)
			var ok bool
			if w.nextENR, ok = e.(*enr.Record); ok {
				w.lock.Unlock()
				w.ns.SetState(w.nextID, w.selected, 0, time.Second*5)
				return true
			}
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

	//	node, _ := enode.New(enode.V4ID{}, w.nextENR)
	node, _ := enode.New(w.validSchemes, w.nextENR)
	return node
}
