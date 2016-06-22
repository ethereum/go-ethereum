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

// Package filters implements an ethereum filtering system for block,
// transactions and log events.
package filters

import (
	"bufio"
	crand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

const (
	// UnknownFilter indicates an unkown filter type
	UnknownFilter Type = iota
	// BlockFilter queries for new blocks
	BlockFilter
	// PendingTxFilter queries pending transactions
	PendingTxFilter
	// LogFilter queries for new or removed (chain reorg) logs
	LogFilter
	// PendingLogFilter queries for logs for the pending block
	PendingLogFilter

	maxPendingHashes = 10240 // max buffer size of block/tx hashes before an filter is considered inactive and disabled
	maxPendingLogs   = 10240 // max buffer size of logs before an filter is considered inactive and disabled
)

var (
	errFilterNotFound           = errors.New("filter not found")
	errCouldNotGenerateFilterID = errors.New("unable to generate filter id")
	errInvalidFilterID          = errors.New("invalid filter id")
	errUnableToUninstallFilter  = errors.New("unable to uninstall filter, retry later")

	filterIDGenMu sync.Mutex
	filterIDGen   = filterIDGenerator()
)

type logsCallback func(FilterID, []Log)

// Type determines the kind of filter and is used to put the filter in to
// the correct bucket when added.
type Type byte

// FilterID determines the type for a filter identifier.
type FilterID [16]byte

// MarshalJSON serializes a FilterID into its JSON representation.
func (f FilterID) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("0x%x", f))
}

// filterIDGenerator helper utility that generates a (pseudo) random sequence of bytes
// that are used to generate filter identifiers.
func filterIDGenerator() *rand.Rand {
	if seed, err := binary.ReadVarint(bufio.NewReader(crand.Reader)); err == nil {
		return rand.New(rand.NewSource(seed))
	}
	return rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
}

// newID generates a filter identifier.
func newID() FilterID {
	filterIDGenMu.Lock()
	defer filterIDGenMu.Unlock()

	var id FilterID
	for i := 0; i < len(id); i += 7 {
		val := filterIDGen.Int63()
		for j := 0; i+j < len(id) && j < 7; j++ {
			id[i+j] = byte(val)
			val >>= 8
		}
	}

	return id
}

// UnmarshalJSON parses a FilterID from its JSON representation.
func (f *FilterID) UnmarshalJSON(data []byte) error {
	// `"0x...."`
	if len(data) != 36 || data[0] != '"' || data[35] != '"' || data[1] != '0' || (data[2] != 'x' && data[2] != 'X') {
		return errInvalidFilterID
	}

	_, err := hex.Decode(f[:], data[3:35])
	return err
}

// Log is a helper that can hold additional information about vm.Log
// necessary for the RPC interface.
type Log struct {
	*vm.Log
	Removed bool `json:"removed"`
}

type filter struct {
	ID         FilterID
	typ        Type
	created    time.Time
	lastUsed   time.Time
	canTimeout bool
	hashes     chan common.Hash // results in case filter type returns hashes
	logsCrit   FilterCriteria
	lc         logsCallback
	logs       chan []Log // results in case filter type returns logs
}

// Manager listens for new events and offers a filter system to queury for
// events that match a set of criteria.
//
// Note, this system cannot be used to query past logs, it will only receive
// and handle events that are posted by the global event mux. Use a raw Filter
// to query for logs that are already stored.
type Manager struct {
	sub event.Subscription

	install   chan *filter // install filter for event notification
	uninstall chan *filter // remove filter for event notification

	allMu sync.RWMutex
	all   map[FilterID]*filter // all installed filters
}

// NewManager creates a new manager that listens for event on the given mux,
// parses and filters them. It uses the all map to retrieve filter changes. The
// work loop holds its own index that is used to forward events to filters.
//
// The returned manager has a loop that needs to be stopped with the Stop function
// or by stopping the given mux.
func NewManager(mux *event.TypeMux) *Manager {
	sub := mux.Subscribe(
		core.PendingLogsEvent{},
		core.RemovedLogsEvent{},
		core.ChainEvent{},
		core.TxPreEvent{},
		vm.Logs(nil),
	)

	m := &Manager{
		sub:       sub,
		install:   make(chan *filter),
		uninstall: make(chan *filter, 1024),
		all:       make(map[FilterID]*filter),
	}

	go m.run()

	return m
}

// Uninstall filter. If the given filter could not be found an error is returned.
func (m *Manager) Uninstall(id FilterID) error {
	m.allMu.Lock()

	if f, found := m.all[id]; found {
		delete(m.all, id)
		m.allMu.Unlock()

		select {
		case m.uninstall <- f:
			return nil
		default:
			// can only happen when there are too many pending uninstall requests
			return errUnableToUninstallFilter
		}
	}

	m.allMu.Unlock()
	return errFilterNotFound
}

// FilterType returns the filter type for the given id.
// If the filter could be found UnknownFilter is returned.
func (m *Manager) FilterType(id FilterID) Type {
	m.allMu.RLock()
	defer m.allMu.RUnlock()

	if f, found := m.all[id]; found {
		return f.typ
	}
	return UnknownFilter
}

// NewBlockFilter returns a filter identifier that can be used to get the hashes
// for new blocks. The given callback is optional. If nil is given block hashes
// are queued until fetched with GetBlockFilterChanges.
func (m *Manager) NewBlockFilter() (FilterID, error) {
	id := newID()

	f := &filter{
		ID:         id,
		created:    time.Now(),
		lastUsed:   time.Now(),
		canTimeout: true,
		typ:        BlockFilter,
		hashes:     make(chan common.Hash, maxPendingHashes),
		logs:       make(chan []Log, maxPendingLogs),
	}

	m.allMu.Lock()
	m.all[id] = f
	m.allMu.Unlock()

	m.install <- f

	return id, nil
}

// GetBlockFilterChanges returns the data for the filter with the given id since
// last time it was called.
func (m *Manager) GetBlockFilterChanges(id FilterID) ([]common.Hash, error) {
	m.allMu.RLock()
	if f, found := m.all[id]; found && f.typ == BlockFilter {
		m.allMu.RUnlock()

		// retieve hashes
		f.lastUsed = time.Now()
		hashes := make([]common.Hash, 0, len(f.hashes)) // prevent (most) allocs
		for {
			select {
			case h := <-f.hashes:
				hashes = append(hashes, h)
			default:
				return hashes, nil
			}
		}
	}

	m.allMu.RUnlock()
	return nil, errFilterNotFound
}

// NewLogFilterWithNoTimeout returns a filter identifier that can be used to fetch
// logs matching the given criteria. The created filter will not timeout and the
// callee is expected to manually uninstall the filter.
func (m *Manager) NewLogFilterWithNoTimeout(crit FilterCriteria, cb logsCallback) (FilterID, error) {
	id := newID()

	f := &filter{
		ID:         id,
		created:    time.Now(),
		lastUsed:   time.Now(),
		canTimeout: false,
		typ:        LogFilter,
		hashes:     make(chan common.Hash, maxPendingHashes),
		logs:       make(chan []Log, maxPendingLogs),
		lc:         cb,
		logsCrit:   crit,
	}

	m.allMu.Lock()
	m.all[id] = f
	m.allMu.Unlock()

	m.install <- f

	return id, nil
}

// NewLogFilter returns a filter identifier that can be used to fetch logs matching
// the given criteria.
func (m *Manager) NewLogFilter(crit FilterCriteria, cb logsCallback) (FilterID, error) {
	id := newID()

	f := &filter{
		ID:         id,
		created:    time.Now(),
		lastUsed:   time.Now(),
		canTimeout: true,
		typ:        LogFilter,
		hashes:     make(chan common.Hash, maxPendingHashes),
		logs:       make(chan []Log, maxPendingLogs),
		lc:         cb,
		logsCrit:   crit,
	}

	m.allMu.Lock()
	m.all[id] = f
	m.allMu.Unlock()

	m.install <- f

	return id, nil
}

// GetLogFilterCriteria returns the filtering criteria for a filter, or an error in
// case the filter could not be found.
func (m *Manager) GetLogFilterCriteria(id FilterID) (FilterCriteria, error) {
	m.allMu.RLock()
	defer m.allMu.RUnlock()

	if f, found := m.all[id]; found {
		return f.logsCrit, nil
	}

	return FilterCriteria{}, errFilterNotFound
}

// NewPendingLogFilter creates a filter that returns new pending logs that match the given criteria.
func (m *Manager) NewPendingLogFilter(crit FilterCriteria, cb logsCallback) (FilterID, error) {
	id := newID()

	f := &filter{
		ID:         id,
		created:    time.Now(),
		lastUsed:   time.Now(),
		canTimeout: true,
		typ:        PendingLogFilter,
		hashes:     make(chan common.Hash, maxPendingHashes),
		logs:       make(chan []Log, maxPendingLogs),
		lc:         cb,
		logsCrit:   crit,
	}

	m.allMu.Lock()
	m.all[id] = f
	m.allMu.Unlock()

	m.install <- f

	return id, nil
}

// GetPendingLogFilterChanges returns logs for the pending block.
func (m *Manager) GetPendingLogFilterChanges(id FilterID) ([]Log, error) {
	m.allMu.RLock()
	defer m.allMu.RUnlock()

	if f, found := m.all[id]; found && f.typ == PendingLogFilter {
		f.lastUsed = time.Now()
		allLogs := make([]Log, 0, len(f.logs)) // prevent (most) allocs
		for {
			select {
			case logs := <-f.logs:
				allLogs = append(allLogs, logs...)
			default: // available logs read
				return allLogs, nil
			}
		}
	}

	return nil, errFilterNotFound
}

// GetLogFilterChanges returns all logs matching the criteria for the filter with the given filter id.
func (m *Manager) GetLogFilterChanges(id FilterID) ([]Log, error) {
	m.allMu.RLock()
	defer m.allMu.RUnlock()

	if f, found := m.all[id]; found && f.typ == LogFilter {
		f.lastUsed = time.Now()
		allLogs := make([]Log, 0, len(f.logs)) // prevent (most) allocs for the append
		for {
			select {
			case logs := <-f.logs:
				allLogs = append(allLogs, logs...)
			default: // available logs read
				return allLogs, nil
			}
		}
	}

	return nil, errFilterNotFound
}

// NewPendingTransactionFilter creates a filter that retrieves pending transactions.
func (m *Manager) NewPendingTransactionFilter() (FilterID, error) {
	id := newID()

	f := &filter{
		ID:         id,
		created:    time.Now(),
		lastUsed:   time.Now(),
		canTimeout: true,
		typ:        PendingTxFilter,
		hashes:     make(chan common.Hash, maxPendingHashes),
		logs:       make(chan []Log, maxPendingLogs),
	}

	m.allMu.Lock()
	m.all[id] = f
	m.allMu.Unlock()

	m.install <- f

	return id, nil
}

// GetPendingTxFilterChanges returns hashes for pending transactions which are added since the last poll.
func (m *Manager) GetPendingTxFilterChanges(id FilterID) ([]common.Hash, error) {
	m.allMu.RLock()
	defer m.allMu.RUnlock()

	if f, found := m.all[id]; found && f.typ == PendingTxFilter {
		f.lastUsed = time.Now()
		hashes := make([]common.Hash, 0, len(f.hashes)) // prevent (most) allocs
		for {
			select {
			case hash := <-f.hashes:
				hashes = append(hashes, hash)
			default: // read available tx hashes
				return hashes, nil
			}
		}
	}

	return nil, errFilterNotFound
}

type filterIndex map[Type]map[FilterID]*filter

// process an event and forward to filters that match the criteria.
func (m *Manager) process(filters filterIndex, ev *event.Event) {
	var inactive []*filter

	logHandler := func(f *filter, logs []Log) {
		if f.lc != nil && len(logs) > 0 {
			f.lc(f.ID, logs)
		} else if len(logs) > 0 {
			select {
			case f.logs <- logs:
				return
			default: // data queue full, disable filter
				inactive = append(inactive, f)
			}
		}
	}

	switch e := ev.Data.(type) {
	case core.ChainEvent:
		for _, f := range filters[BlockFilter] {
			if ev.Time.After(f.created) {
				select {
				case f.hashes <- e.Hash:
					continue
				default:
					// data queue full, disable filter
					inactive = append(inactive, f)
				}
			}
		}
	case core.TxPreEvent:
		for _, f := range filters[PendingTxFilter] {
			if ev.Time.After(f.created) {
				select {
				case f.hashes <- e.Tx.Hash():
					continue
				default:
					// data queue full, disable filter
					inactive = append(inactive, f)
				}
			}
		}
	case vm.Logs:
		for _, f := range filters[LogFilter] {
			if ev.Time.After(f.created) {
				matchedLogs := filterLogs(convertLogs(e, false), f.logsCrit.Addresses, f.logsCrit.Topics)
				logHandler(f, matchedLogs)
			}
		}
	case core.RemovedLogsEvent:
		for _, f := range filters[LogFilter] {
			if ev.Time.After(f.created) {
				matchedLogs := filterLogs(convertLogs(e.Logs, true), f.logsCrit.Addresses, f.logsCrit.Topics)
				logHandler(f, matchedLogs)
			}
		}
	case core.PendingLogsEvent:
		for _, f := range filters[PendingLogFilter] {
			if ev.Time.After(f.created) {
				matchedLogs := filterLogs(convertLogs(e.Logs, false), f.logsCrit.Addresses, f.logsCrit.Topics)
				logHandler(f, matchedLogs)
			}
		}
	}

	m.allMu.Lock()
	for _, f := range inactive {
		delete(m.all, f.ID)
		glog.Warningf("filter 0x%x uninstalled, queue full\n", f.ID)
	}
	m.allMu.Unlock()

	// remove filter for event listening, this must be run in a seperate go routine
	// since timeout is called from the work loop and sending uninstall requests to the
	// work loop from "itself" may deadlock when the uninstall channel is full.
	go func() {
		for _, f := range inactive {
			m.uninstall <- f
		}
	}()
}

// timeout uninstalls all filters that have not been used in the last 5 minutes.
func (m *Manager) timeout() {
	deadline := time.Now().Add(-5 * time.Minute)
	var inactive []*filter
	m.allMu.Lock()
	for _, f := range m.all {
		if f.lastUsed.Before(deadline) && f.canTimeout {
			delete(m.all, f.ID) // filter cannot be used from the external
			inactive = append(inactive, f)
		}
	}
	m.allMu.Unlock()

	// remove filter for event listening, this must be run in a seperate go routine
	// since timeout is called from the work loop and sending uninstall requests to the
	// work loop from "itself" may deadlock when the uninstall channel is full.
	go func() {
		for _, f := range inactive {
			m.uninstall <- f // delete for the internal work loop
		}
	}()
}

// run is the manager work loop.
// It will receive events and forwards them to installed filters.
// Inactive filters will be uninstalled.
func (m *Manager) run() {
	index := make(filterIndex)
	timeout := time.NewTicker(30 * time.Second)

	for {
		select {
		case f := <-m.install:
			// lazy load
			if _, found := index[f.typ]; !found {
				index[f.typ] = make(map[FilterID]*filter)
			}
			index[f.typ][f.ID] = f
		case f := <-m.uninstall:
			close(f.hashes)
			close(f.logs)
			delete(index[f.typ], f.ID)
		case ev, ok := <-m.sub.Chan():
			if !ok {
				glog.V(logger.Debug).Infoln("filter manager stopped")
				return
			}
			m.process(index, ev)
		case <-timeout.C:
			m.timeout()
		}
	}
}

// Stop the filter system.
func (m *Manager) Stop() {
	m.sub.Unsubscribe() // end worker loop
}

// convertLogs is a helper utility that converts vm.Logs to []filter.Log.
func convertLogs(in vm.Logs, removed bool) []Log {
	logs := make([]Log, len(in))
	for i, l := range in {
		logs[i] = Log{l, false}
	}
	return logs
}
