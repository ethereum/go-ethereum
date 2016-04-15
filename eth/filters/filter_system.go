// Copyright 2015 The go-ethereum Authors
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
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/event"
)

// FilterType determines the type of filter and is used to put the filter in to
// the correct bucket when added.
type FilterType byte

const (
	ChainFilter      FilterType = iota // new block events filter
	PendingTxFilter                    // pending transaction filter
	LogFilter                          // new or removed log filter
	PendingLogFilter                   // pending log filter
)

// FilterSystem manages filters that filter specific events such as
// block, transaction and log events. The Filtering system can be used to listen
// for specific LOG events fired by the EVM (Ethereum Virtual Machine).
type FilterSystem struct {
	filterMu sync.RWMutex
	filterId int

	chainFilters      map[int]*Filter
	pendingTxFilters  map[int]*Filter
	logFilters        map[int]*Filter
	pendingLogFilters map[int]*Filter

	// generic is an ugly hack for Get
	generic map[int]*Filter

	sub event.Subscription
}

// NewFilterSystem returns a newly allocated filter manager
func NewFilterSystem(mux *event.TypeMux) *FilterSystem {
	fs := &FilterSystem{
		chainFilters:      make(map[int]*Filter),
		pendingTxFilters:  make(map[int]*Filter),
		logFilters:        make(map[int]*Filter),
		pendingLogFilters: make(map[int]*Filter),
		generic:           make(map[int]*Filter),
	}
	fs.sub = mux.Subscribe(
		core.PendingLogsEvent{},
		core.RemovedLogsEvent{},
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
func (fs *FilterSystem) Add(filter *Filter, filterType FilterType) (int, error) {
	fs.filterMu.Lock()
	defer fs.filterMu.Unlock()

	id := fs.filterId
	filter.created = time.Now()

	switch filterType {
	case ChainFilter:
		fs.chainFilters[id] = filter
	case PendingTxFilter:
		fs.pendingTxFilters[id] = filter
	case LogFilter:
		fs.logFilters[id] = filter
	case PendingLogFilter:
		fs.pendingLogFilters[id] = filter
	default:
		return 0, fmt.Errorf("unknown filter type %v", filterType)
	}
	fs.generic[id] = filter

	fs.filterId++

	return id, nil
}

// Remove removes a filter by filter id
func (fs *FilterSystem) Remove(id int) {
	fs.filterMu.Lock()
	defer fs.filterMu.Unlock()

	delete(fs.chainFilters, id)
	delete(fs.pendingTxFilters, id)
	delete(fs.logFilters, id)
	delete(fs.pendingLogFilters, id)
	delete(fs.generic, id)
}

func (fs *FilterSystem) Get(id int) *Filter {
	fs.filterMu.RLock()
	defer fs.filterMu.RUnlock()

	return fs.generic[id]
}

// filterLoop waits for specific events from ethereum and fires their handlers
// when the filter matches the requirements.
func (fs *FilterSystem) filterLoop() {
	for event := range fs.sub.Chan() {
		switch ev := event.Data.(type) {
		case core.ChainEvent:
			fs.filterMu.RLock()
			for _, filter := range fs.chainFilters {
				if filter.BlockCallback != nil && !filter.created.After(event.Time) {
					filter.BlockCallback(ev.Block, ev.Logs)
				}
			}
			fs.filterMu.RUnlock()
		case core.TxPreEvent:
			fs.filterMu.RLock()
			for _, filter := range fs.pendingTxFilters {
				if filter.TransactionCallback != nil && !filter.created.After(event.Time) {
					filter.TransactionCallback(ev.Tx)
				}
			}
			fs.filterMu.RUnlock()

		case vm.Logs:
			fs.filterMu.RLock()
			for _, filter := range fs.logFilters {
				if filter.LogCallback != nil && !filter.created.After(event.Time) {
					for _, log := range filter.FilterLogs(ev) {
						filter.LogCallback(log, false)
					}
				}
			}
			fs.filterMu.RUnlock()
		case core.RemovedLogsEvent:
			fs.filterMu.RLock()
			for _, filter := range fs.logFilters {
				if filter.LogCallback != nil && !filter.created.After(event.Time) {
					for _, removedLog := range ev.Logs {
						filter.LogCallback(removedLog, true)
					}
				}
			}
			fs.filterMu.RUnlock()
		case core.PendingLogsEvent:
			fs.filterMu.RLock()
			for _, filter := range fs.pendingLogFilters {
				if filter.LogCallback != nil && !filter.created.After(event.Time) {
					for _, pendingLog := range ev.Logs {
						filter.LogCallback(pendingLog, false)
					}
				}
			}
			fs.filterMu.RUnlock()
		}
	}
}
