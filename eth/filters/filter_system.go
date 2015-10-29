// Copyright 2014 The go-ethereum Authors
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

// package filters implements an ethereum filtering system for block,
// transactions and log events.
package filters

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/event"
)

// FilterSystem manages filters that filter specific events such as
// block, transaction and log events. The Filtering system can be used to listen
// for specific LOG events fired by the EVM (Ethereum Virtual Machine).
type FilterSystem struct {
	filterMu sync.RWMutex
	filterId int
	filters  map[int]*Filter
	created  map[int]time.Time
	sub      event.Subscription
}

// NewFilterSystem returns a newly allocated filter manager
func NewFilterSystem(mux *event.TypeMux) *FilterSystem {
	fs := &FilterSystem{
		filters: make(map[int]*Filter),
		created: make(map[int]time.Time),
	}
	fs.sub = mux.Subscribe(
		//core.PendingBlockEvent{},
		core.ChainEvent{},
		core.TxPreEvent{},
		vm.Logs(nil),
	)
	go fs.filterLoop()
	return fs
}

// Stop quits the filter loop required for polling events
func (fs *FilterSystem) Stop() {
	fs.sub.Unsubscribe()
}

// Add adds a filter to the filter manager
func (fs *FilterSystem) Add(filter *Filter) (id int) {
	fs.filterMu.Lock()
	defer fs.filterMu.Unlock()
	id = fs.filterId
	fs.filters[id] = filter
	fs.created[id] = time.Now()
	fs.filterId++

	return id
}

// Remove removes a filter by filter id
func (fs *FilterSystem) Remove(id int) {
	fs.filterMu.Lock()
	defer fs.filterMu.Unlock()

	delete(fs.filters, id)
	delete(fs.created, id)
}

// Get retrieves a filter installed using Add The filter may not be modified.
func (fs *FilterSystem) Get(id int) *Filter {
	fs.filterMu.RLock()
	defer fs.filterMu.RUnlock()

	return fs.filters[id]
}

// filterLoop waits for specific events from ethereum and fires their handlers
// when the filter matches the requirements.
func (fs *FilterSystem) filterLoop() {
	for event := range fs.sub.Chan() {
		switch ev := event.Data.(type) {
		case core.ChainEvent:
			fs.filterMu.RLock()
			for id, filter := range fs.filters {
				if filter.BlockCallback != nil && fs.created[id].Before(event.Time) {
					filter.BlockCallback(ev.Block, ev.Logs)
				}
			}
			fs.filterMu.RUnlock()

		case core.TxPreEvent:
			fs.filterMu.RLock()
			for id, filter := range fs.filters {
				if filter.TransactionCallback != nil && fs.created[id].Before(event.Time) {
					filter.TransactionCallback(ev.Tx)
				}
			}
			fs.filterMu.RUnlock()

		case vm.Logs:
			fs.filterMu.RLock()
			for id, filter := range fs.filters {
				if filter.LogsCallback != nil && fs.created[id].Before(event.Time) {
					msgs := filter.FilterLogs(ev)
					if len(msgs) > 0 {
						filter.LogsCallback(msgs)
					}
				}
			}
			fs.filterMu.RUnlock()
		}
	}
}
