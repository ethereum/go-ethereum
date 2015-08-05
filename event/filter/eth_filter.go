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

package filter

// TODO make use of the generic filtering system

import (
	"sync"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/event"
)

type FilterManager struct {
	eventMux *event.TypeMux

	filterMu sync.RWMutex
	filterId int
	filters  map[int]*core.Filter

	quit chan struct{}
}

func NewFilterManager(mux *event.TypeMux) *FilterManager {
	return &FilterManager{
		eventMux: mux,
		filters:  make(map[int]*core.Filter),
	}
}

func (self *FilterManager) Start() {
	go self.filterLoop()
}

func (self *FilterManager) Stop() {
	close(self.quit)
}

func (self *FilterManager) InstallFilter(filter *core.Filter) (id int) {
	self.filterMu.Lock()
	defer self.filterMu.Unlock()
	id = self.filterId
	self.filters[id] = filter
	self.filterId++

	return id
}

func (self *FilterManager) UninstallFilter(id int) {
	self.filterMu.Lock()
	defer self.filterMu.Unlock()
	if _, ok := self.filters[id]; ok {
		delete(self.filters, id)
	}
}

// GetFilter retrieves a filter installed using InstallFilter.
// The filter may not be modified.
func (self *FilterManager) GetFilter(id int) *core.Filter {
	self.filterMu.RLock()
	defer self.filterMu.RUnlock()
	return self.filters[id]
}

func (self *FilterManager) filterLoop() {
	// Subscribe to events
	events := self.eventMux.Subscribe(
		//core.PendingBlockEvent{},
		core.ChainEvent{},
		core.TxPreEvent{},
		state.Logs(nil))

out:
	for {
		select {
		case <-self.quit:
			break out
		case event := <-events.Chan():
			switch event := event.(type) {
			case core.ChainEvent:
				self.filterMu.RLock()
				for _, filter := range self.filters {
					if filter.BlockCallback != nil {
						filter.BlockCallback(event.Block, event.Logs)
					}
				}
				self.filterMu.RUnlock()

			case core.TxPreEvent:
				self.filterMu.RLock()
				for _, filter := range self.filters {
					if filter.TransactionCallback != nil {
						filter.TransactionCallback(event.Tx)
					}
				}
				self.filterMu.RUnlock()

			case state.Logs:
				self.filterMu.RLock()
				for _, filter := range self.filters {
					if filter.LogsCallback != nil {
						msgs := filter.FilterLogs(event)
						if len(msgs) > 0 {
							filter.LogsCallback(msgs)
						}
					}
				}
				self.filterMu.RUnlock()
			}
		}
	}
}
