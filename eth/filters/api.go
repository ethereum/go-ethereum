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

package filters

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	deadline = 5 * time.Minute // consider a filter inactive if it has not been polled for within deadline
)

// filter is a helper struct that holds meta information over the filter type
// and associated subscription in the event system.
type filter struct {
	typ      Type
	deadline *time.Timer // filter is inactiv when deadline triggers
	hashes   []common.Hash
	crit     FilterCriteria
	logs     []Log
	s        *Subscription // associated subscription in event system
}

// PublicFilterAPI offers support to create and manage filters. This will allow external clients to retrieve various
// information related to the Ethereum protocol such als blocks, transactions and logs.
type PublicFilterAPI struct {
	mux       *event.TypeMux
	quit      chan struct{}
	chainDb   ethdb.Database
	events    *EventSystem
	filtersMu sync.Mutex
	filters   map[rpc.ID]*filter
}

// NewPublicFilterAPI returns a new PublicFilterAPI instance.
func NewPublicFilterAPI(chainDb ethdb.Database, mux *event.TypeMux) *PublicFilterAPI {
	api := &PublicFilterAPI{
		mux:     mux,
		chainDb: chainDb,
		events:  NewEventSystem(mux),
		filters: make(map[rpc.ID]*filter),
	}

	go api.timeoutLoop()

	return api
}

// timeoutLoop runs every 5 minutes and deletes filters that have not been recently used.
// Tt is started when the api is created.
func (api *PublicFilterAPI) timeoutLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for {
		<-ticker.C
		api.filtersMu.Lock()
		for id, f := range api.filters {
			select {
			case <-f.deadline.C:
				f.s.Unsubscribe()
				delete(api.filters, id)
			default:
				continue
			}
		}
		api.filtersMu.Unlock()
	}
}

// NewPendingTransactionFilter creates a filter that fetches pending transaction hashes
// as transactions enter the pending state.
//
// It is part of the filter package because this filter can be used throug the
// `eth_getFilterChanges` polling method that is also used for log filters.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newpendingtransactionfilter
func (api *PublicFilterAPI) NewPendingTransactionFilter() rpc.ID {
	var (
		pendingTxs   = make(chan common.Hash)
		pendingTxSub = api.events.SubscribePendingTxEvents(pendingTxs)
	)

	api.filtersMu.Lock()
	api.filters[pendingTxSub.ID] = &filter{typ: PendingTransactionsSubscription, deadline: time.NewTimer(deadline), hashes: make([]common.Hash, 0), s: pendingTxSub}
	api.filtersMu.Unlock()

	go func() {
		for {
			select {
			case ph := <-pendingTxs:
				api.filtersMu.Lock()
				if f, found := api.filters[pendingTxSub.ID]; found {
					f.hashes = append(f.hashes, ph)
				}
				api.filtersMu.Unlock()
			case <-pendingTxSub.Err():
				api.filtersMu.Lock()
				delete(api.filters, pendingTxSub.ID)
				api.filtersMu.Unlock()
				return
			}
		}
	}()

	return pendingTxSub.ID
}

// NewPendingTransactions creates a subscription that is triggered each time a transaction
// enters the transaction pool and was signed from one of the transactions this nodes manages.
func (api *PublicFilterAPI) NewPendingTransactions(ctx context.Context) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		txHashes := make(chan common.Hash)
		pendingTxSub := api.events.SubscribePendingTxEvents(txHashes)

		for {
			select {
			case h := <-txHashes:
				notifier.Notify(rpcSub.ID, h)
			case <-rpcSub.Err():
				pendingTxSub.Unsubscribe()
				return
			case <-notifier.Closed():
				pendingTxSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}

// NewBlockFilter creates a filter that fetches blocks that are imported into the chain.
// It is part of the filter package since polling goes with eth_getFilterChanges.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newblockfilter
func (api *PublicFilterAPI) NewBlockFilter() rpc.ID {
	var (
		headers   = make(chan *types.Header)
		headerSub = api.events.SubscribeNewHeads(headers)
	)

	api.filtersMu.Lock()
	api.filters[headerSub.ID] = &filter{typ: BlocksSubscription, deadline: time.NewTimer(deadline), hashes: make([]common.Hash, 0), s: headerSub}
	api.filtersMu.Unlock()

	go func() {
		for {
			select {
			case h := <-headers:
				api.filtersMu.Lock()
				if f, found := api.filters[headerSub.ID]; found {
					f.hashes = append(f.hashes, h.Hash())
				}
				api.filtersMu.Unlock()
			case <-headerSub.Err():
				api.filtersMu.Lock()
				delete(api.filters, headerSub.ID)
				api.filtersMu.Unlock()
				return
			}
		}
	}()

	return headerSub.ID
}

// NewHeads send a notification each time a new (header) block is appended to the chain.
func (api *PublicFilterAPI) NewHeads(ctx context.Context) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		headers := make(chan *types.Header)
		headersSub := api.events.SubscribeNewHeads(headers)

		for {
			select {
			case h := <-headers:
				notifier.Notify(rpcSub.ID, h)
			case <-rpcSub.Err():
				headersSub.Unsubscribe()
				return
			case <-notifier.Closed():
				headersSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}

// Logs creates a subscription that fires for all new log that match the given filter criteria.
func (api *PublicFilterAPI) Logs(ctx context.Context, crit FilterCriteria) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		matchedLogs := make(chan []Log)
		logsSub := api.events.SubscribeLogs(crit, matchedLogs)

		for {
			select {
			case logs := <-matchedLogs:
				for _, log := range logs {
					notifier.Notify(rpcSub.ID, &log)
				}
			case <-rpcSub.Err(): // client send an unsubscribe request
				logsSub.Unsubscribe()
				return
			case <-notifier.Closed(): // connection dropped
				logsSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}

// FilterCriteria represents a request to create a new filter.
type FilterCriteria struct {
	FromBlock *big.Int
	ToBlock   *big.Int
	Addresses []common.Address
	Topics    [][]common.Hash
}

// NewFilter creates a new filter and returns the filter id. It can be
// used to retrieve logs when the state changes. This method cannot be
// used to fetch logs that are already stored in the state.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newfilter
func (api *PublicFilterAPI) NewFilter(crit FilterCriteria) rpc.ID {
	var (
		logs    = make(chan []Log)
		logsSub = api.events.SubscribeLogs(crit, logs)
	)

	if crit.FromBlock == nil {
		crit.FromBlock = big.NewInt(rpc.LatestBlockNumber.Int64())
	}
	if crit.ToBlock == nil {
		crit.ToBlock = big.NewInt(rpc.LatestBlockNumber.Int64())
	}

	api.filtersMu.Lock()
	api.filters[logsSub.ID] = &filter{typ: LogsSubscription, crit: crit, deadline: time.NewTimer(deadline), logs: make([]Log, 0), s: logsSub}
	api.filtersMu.Unlock()

	go func() {
		for {
			select {
			case l := <-logs:
				api.filtersMu.Lock()
				if f, found := api.filters[logsSub.ID]; found {
					f.logs = append(f.logs, l...)
				}
				api.filtersMu.Unlock()
			case <-logsSub.Err():
				api.filtersMu.Lock()
				delete(api.filters, logsSub.ID)
				api.filtersMu.Unlock()
				return
			}
		}
	}()

	return logsSub.ID
}

// GetLogs returns logs matching the given argument that are stored within the state.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getlogs
func (api *PublicFilterAPI) GetLogs(crit FilterCriteria) []Log {
	if crit.FromBlock == nil {
		crit.FromBlock = big.NewInt(rpc.LatestBlockNumber.Int64())
	}
	if crit.ToBlock == nil {
		crit.ToBlock = big.NewInt(rpc.LatestBlockNumber.Int64())
	}

	filter := New(api.chainDb)
	filter.SetBeginBlock(crit.FromBlock.Int64())
	filter.SetEndBlock(crit.ToBlock.Int64())
	filter.SetAddresses(crit.Addresses)
	filter.SetTopics(crit.Topics)

	return returnLogs(filter.Find())
}

// UninstallFilter removes the filter with the given filter id.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_uninstallfilter
func (api *PublicFilterAPI) UninstallFilter(id rpc.ID) bool {
	api.filtersMu.Lock()
	f, found := api.filters[id]
	if found {
		delete(api.filters, id)
	}
	api.filtersMu.Unlock()
	if found {
		f.s.Unsubscribe()
	}

	return found
}

// GetFilterLogs returns the logs for the filter with the given id.
// If the filter could not be found an empty array of logs is returned.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getfilterlogs
func (api *PublicFilterAPI) GetFilterLogs(id rpc.ID) []Log {
	api.filtersMu.Lock()
	f, found := api.filters[id]
	api.filtersMu.Unlock()

	if !found || f.typ != LogsSubscription {
		return []Log{}
	}

	filter := New(api.chainDb)
	filter.SetBeginBlock(f.crit.FromBlock.Int64())
	filter.SetEndBlock(f.crit.ToBlock.Int64())
	filter.SetAddresses(f.crit.Addresses)
	filter.SetTopics(f.crit.Topics)

	return returnLogs(filter.Find())
}

// GetFilterChanges returns the logs for the filter with the given id since
// last time is was called. This can be used for polling.
//
// For pending transaction and block filters the result is []common.Hash.
// (pending)Log filters return []Log. If the filter could not be found
// []interface{}{} is returned.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getfilterchanges
func (api *PublicFilterAPI) GetFilterChanges(id rpc.ID) interface{} {
	api.filtersMu.Lock()
	defer api.filtersMu.Unlock()

	if f, found := api.filters[id]; found {
		if !f.deadline.Stop() {
			// timer expired but filter is not yet removed in timeout loop
			// receive timer value and reset timer
			<-f.deadline.C
		}
		f.deadline.Reset(deadline)

		switch f.typ {
		case PendingTransactionsSubscription, BlocksSubscription:
			hashes := f.hashes
			f.hashes = nil
			return returnHashes(hashes)
		case PendingLogsSubscription, LogsSubscription:
			logs := f.logs
			f.logs = nil
			return returnLogs(logs)
		}
	}

	return []interface{}{}
}

// returnHashes is a helper that will return an empty hash array case the given hash array is nil,
// otherwise the given hashes array is returned.
func returnHashes(hashes []common.Hash) []common.Hash {
	if hashes == nil {
		return []common.Hash{}
	}
	return hashes
}

// returnLogs is a helper that will return an empty log array in case the given logs array is nil,
// otherwise the given logs array is returned.
func returnLogs(logs []Log) []Log {
	if logs == nil {
		return []Log{}
	}
	return logs
}

// UnmarshalJSON sets *args fields with given data.
func (args *FilterCriteria) UnmarshalJSON(data []byte) error {
	type input struct {
		From      *rpc.BlockNumber `json:"fromBlock"`
		ToBlock   *rpc.BlockNumber `json:"toBlock"`
		Addresses interface{}      `json:"address"`
		Topics    []interface{}    `json:"topics"`
	}

	var raw input
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if raw.From == nil || raw.From.Int64() < 0 {
		args.FromBlock = big.NewInt(rpc.LatestBlockNumber.Int64())
	} else {
		args.FromBlock = big.NewInt(raw.From.Int64())
	}

	if raw.ToBlock == nil || raw.ToBlock.Int64() < 0 {
		args.ToBlock = big.NewInt(rpc.LatestBlockNumber.Int64())
	} else {
		args.ToBlock = big.NewInt(raw.ToBlock.Int64())
	}

	args.Addresses = []common.Address{}

	if raw.Addresses != nil {
		// raw.Address can contain a single address or an array of addresses
		var addresses []common.Address
		if strAddrs, ok := raw.Addresses.([]interface{}); ok {
			for i, addr := range strAddrs {
				if strAddr, ok := addr.(string); ok {
					if len(strAddr) >= 2 && strAddr[0] == '0' && (strAddr[1] == 'x' || strAddr[1] == 'X') {
						strAddr = strAddr[2:]
					}
					if decAddr, err := hex.DecodeString(strAddr); err == nil {
						addresses = append(addresses, common.BytesToAddress(decAddr))
					} else {
						return fmt.Errorf("invalid address given")
					}
				} else {
					return fmt.Errorf("invalid address on index %d", i)
				}
			}
		} else if singleAddr, ok := raw.Addresses.(string); ok {
			if len(singleAddr) >= 2 && singleAddr[0] == '0' && (singleAddr[1] == 'x' || singleAddr[1] == 'X') {
				singleAddr = singleAddr[2:]
			}
			if decAddr, err := hex.DecodeString(singleAddr); err == nil {
				addresses = append(addresses, common.BytesToAddress(decAddr))
			} else {
				return fmt.Errorf("invalid address given")
			}
		} else {
			return errors.New("invalid address(es) given")
		}
		args.Addresses = addresses
	}

	// helper function which parses a string to a topic hash
	topicConverter := func(raw string) (common.Hash, error) {
		if len(raw) == 0 {
			return common.Hash{}, nil
		}
		if len(raw) >= 2 && raw[0] == '0' && (raw[1] == 'x' || raw[1] == 'X') {
			raw = raw[2:]
		}
		if len(raw) != 2*common.HashLength {
			return common.Hash{}, errors.New("invalid topic(s)")
		}
		if decAddr, err := hex.DecodeString(raw); err == nil {
			return common.BytesToHash(decAddr), nil
		}
		return common.Hash{}, errors.New("invalid topic(s)")
	}

	// topics is an array consisting of strings and/or arrays of strings.
	// JSON null values are converted to common.Hash{} and ignored by the filter manager.
	if len(raw.Topics) > 0 {
		args.Topics = make([][]common.Hash, len(raw.Topics))
		for i, t := range raw.Topics {
			if t == nil { // ignore topic when matching logs
				args.Topics[i] = []common.Hash{common.Hash{}}
			} else if topic, ok := t.(string); ok { // match specific topic
				top, err := topicConverter(topic)
				if err != nil {
					return err
				}
				args.Topics[i] = []common.Hash{top}
			} else if topics, ok := t.([]interface{}); ok { // or case e.g. [null, "topic0", "topic1"]
				for _, rawTopic := range topics {
					if rawTopic == nil {
						args.Topics[i] = append(args.Topics[i], common.Hash{})
					} else if topic, ok := rawTopic.(string); ok {
						parsed, err := topicConverter(topic)
						if err != nil {
							return err
						}
						args.Topics[i] = append(args.Topics[i], parsed)
					} else {
						return fmt.Errorf("invalid topic(s)")
					}
				}
			} else {
				return fmt.Errorf("invalid topic(s)")
			}
		}
	}

	return nil
}
