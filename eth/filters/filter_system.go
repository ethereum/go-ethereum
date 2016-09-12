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
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
)

// Type determines the kind of filter and is used to put the filter in to
// the correct bucket when added.
type Type byte

const (
	// UnknownSubscription indicates an unkown subscription type
	UnknownSubscription Type = iota
	// LogsSubscription queries for new or removed (chain reorg) logs
	LogsSubscription
	// PendingLogsSubscription queries for logs for the pending block
	PendingLogsSubscription
	// PendingTransactionsSubscription queries tx hashes for pending
	// transactions entering the pending state
	PendingTransactionsSubscription
	// BlocksSubscription queries hashes for blocks that are imported
	BlocksSubscription
)

var (
	ErrInvalidSubscriptionID = errors.New("invalid id")
)

// Log is a helper that can hold additional information about vm.Log
// necessary for the RPC interface.
type Log struct {
	*vm.Log
	Removed bool `json:"removed"`
}

func (l *Log) MarshalJSON() ([]byte, error) {
	fields := map[string]interface{}{
		"address":          l.Address,
		"data":             fmt.Sprintf("0x%x", l.Data),
		"blockNumber":      fmt.Sprintf("%#x", l.BlockNumber),
		"logIndex":         fmt.Sprintf("%#x", l.Index),
		"blockHash":        l.BlockHash,
		"transactionHash":  l.TxHash,
		"transactionIndex": fmt.Sprintf("%#x", l.TxIndex),
		"topics":           l.Topics,
		"removed":          l.Removed,
	}

	return json.Marshal(fields)
}

type subscription struct {
	id        rpc.ID
	typ       Type
	created   time.Time
	logsCrit  FilterCriteria
	logs      chan []Log
	hashes    chan common.Hash
	headers   chan *types.Header
	installed chan struct{} // closed when the filter is installed
	err       chan error    // closed when the filter is uninstalled
}

// EventSystem creates subscriptions, processes events and broadcasts them to the
// subscription which match the subscription criteria.
type EventSystem struct {
	mux       *event.TypeMux
	sub       event.Subscription
	install   chan *subscription // install filter for event notification
	uninstall chan *subscription // remove filter for event notification
}

// NewEventSystem creates a new manager that listens for event on the given mux,
// parses and filters them. It uses the all map to retrieve filter changes. The
// work loop holds its own index that is used to forward events to filters.
//
// The returned manager has a loop that needs to be stopped with the Stop function
// or by stopping the given mux.
func NewEventSystem(mux *event.TypeMux) *EventSystem {
	m := &EventSystem{
		mux:       mux,
		install:   make(chan *subscription),
		uninstall: make(chan *subscription),
	}

	go m.eventLoop()

	return m
}

// Subscription is created when the client registers itself for a particular event.
type Subscription struct {
	ID        rpc.ID
	f         *subscription
	es        *EventSystem
	unsubOnce sync.Once
}

// Err returns a channel that is closed when unsubscribed.
func (sub *Subscription) Err() <-chan error {
	return sub.f.err
}

// Unsubscribe uninstalls the subscription from the event broadcast loop.
func (sub *Subscription) Unsubscribe() {
	sub.unsubOnce.Do(func() {
	uninstallLoop:
		for {
			// write uninstall request and consume logs/hashes. This prevents
			// the eventLoop broadcast method to deadlock when writing to the
			// filter event channel while the subscription loop is waiting for
			// this method to return (and thus not reading these events).
			select {
			case sub.es.uninstall <- sub.f:
				break uninstallLoop
			case <-sub.f.logs:
			case <-sub.f.hashes:
			case <-sub.f.headers:
			}
		}

		// wait for filter to be uninstalled in work loop before returning
		// this ensures that the manager won't use the event channel which
		// will probably be closed by the client asap after this method returns.
		<-sub.Err()
	})
}

// subscribe installs the subscription in the event broadcast loop.
func (es *EventSystem) subscribe(sub *subscription) *Subscription {
	es.install <- sub
	<-sub.installed
	return &Subscription{ID: sub.id, f: sub, es: es}
}

// SubscribeLogs creates a subscription that will write all logs matching the
// given criteria to the given logs channel.
func (es *EventSystem) SubscribeLogs(crit FilterCriteria, logs chan []Log) *Subscription {
	sub := &subscription{
		id:        rpc.NewID(),
		typ:       LogsSubscription,
		logsCrit:  crit,
		created:   time.Now(),
		logs:      logs,
		hashes:    make(chan common.Hash),
		headers:   make(chan *types.Header),
		installed: make(chan struct{}),
		err:       make(chan error),
	}

	return es.subscribe(sub)
}

// SubscribePendingLogs creates a subscription that will write pending logs matching the
// given criteria to the given channel.
func (es *EventSystem) SubscribePendingLogs(crit FilterCriteria, logs chan []Log) *Subscription {
	sub := &subscription{
		id:        rpc.NewID(),
		typ:       PendingLogsSubscription,
		logsCrit:  crit,
		created:   time.Now(),
		logs:      logs,
		hashes:    make(chan common.Hash),
		headers:   make(chan *types.Header),
		installed: make(chan struct{}),
		err:       make(chan error),
	}

	return es.subscribe(sub)
}

// SubscribePendingTxEvents creates a sbuscription that writes transaction hashes for
// transactions that enter the transaction pool.
func (es *EventSystem) SubscribePendingTxEvents(hashes chan common.Hash) *Subscription {
	sub := &subscription{
		id:        rpc.NewID(),
		typ:       PendingTransactionsSubscription,
		created:   time.Now(),
		logs:      make(chan []Log),
		hashes:    hashes,
		headers:   make(chan *types.Header),
		installed: make(chan struct{}),
		err:       make(chan error),
	}

	return es.subscribe(sub)
}

// SubscribeNewHeads creates a subscription that writes the header of a block that is
// imported in the chain.
func (es *EventSystem) SubscribeNewHeads(headers chan *types.Header) *Subscription {
	sub := &subscription{
		id:        rpc.NewID(),
		typ:       BlocksSubscription,
		created:   time.Now(),
		logs:      make(chan []Log),
		hashes:    make(chan common.Hash),
		headers:   headers,
		installed: make(chan struct{}),
		err:       make(chan error),
	}

	return es.subscribe(sub)
}

type filterIndex map[Type]map[rpc.ID]*subscription

// broadcast event to filters that match criteria.
func broadcast(filters filterIndex, ev *event.Event) {
	if ev == nil {
		return
	}

	switch e := ev.Data.(type) {
	case vm.Logs:
		if len(e) > 0 {
			for _, f := range filters[LogsSubscription] {
				if ev.Time.After(f.created) {
					if matchedLogs := filterLogs(convertLogs(e, false), f.logsCrit.Addresses, f.logsCrit.Topics); len(matchedLogs) > 0 {
						f.logs <- matchedLogs
					}
				}
			}
		}
	case core.RemovedLogsEvent:
		for _, f := range filters[LogsSubscription] {
			if ev.Time.After(f.created) {
				if matchedLogs := filterLogs(convertLogs(e.Logs, true), f.logsCrit.Addresses, f.logsCrit.Topics); len(matchedLogs) > 0 {
					f.logs <- matchedLogs
				}
			}
		}
	case core.PendingLogsEvent:
		for _, f := range filters[PendingLogsSubscription] {
			if ev.Time.After(f.created) {
				if matchedLogs := filterLogs(convertLogs(e.Logs, false), f.logsCrit.Addresses, f.logsCrit.Topics); len(matchedLogs) > 0 {
					f.logs <- matchedLogs
				}
			}
		}
	case core.TxPreEvent:
		for _, f := range filters[PendingTransactionsSubscription] {
			if ev.Time.After(f.created) {
				f.hashes <- e.Tx.Hash()
			}
		}
	case core.ChainEvent:
		for _, f := range filters[BlocksSubscription] {
			if ev.Time.After(f.created) {
				f.headers <- e.Block.Header()
			}
		}
	}
}

// eventLoop (un)installs filters and processes mux events.
func (es *EventSystem) eventLoop() {
	var (
		index = make(filterIndex)
		sub   = es.mux.Subscribe(core.PendingLogsEvent{}, core.RemovedLogsEvent{}, vm.Logs{}, core.TxPreEvent{}, core.ChainEvent{})
	)
	for {
		select {
		case ev, active := <-sub.Chan():
			if !active { // system stopped
				return
			}
			broadcast(index, ev)
		case f := <-es.install:
			if _, found := index[f.typ]; !found {
				index[f.typ] = make(map[rpc.ID]*subscription)
			}
			index[f.typ][f.id] = f
			close(f.installed)
		case f := <-es.uninstall:
			delete(index[f.typ], f.id)
			close(f.err)
		}
	}
}

// convertLogs is a helper utility that converts vm.Logs to []filter.Log.
func convertLogs(in vm.Logs, removed bool) []Log {
	logs := make([]Log, len(in))
	for i, l := range in {
		logs[i] = Log{l, removed}
	}
	return logs
}
