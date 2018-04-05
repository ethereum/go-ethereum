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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
	logs     []*types.Log
	s        *Subscription // associated subscription in event system
}

// PublicFilterAPI offers support to create and manage filters. This will allow external clients to retrieve various
// information related to the Ethereum protocol such als blocks, transactions and logs.
type PublicFilterAPI struct {
	backend   Backend
	mux       *event.TypeMux
	quit      chan struct{}
	chainDb   ethdb.Database
	events    *EventSystem
	filtersMu sync.Mutex
	filters   map[rpc.ID]*filter
}

// NewPublicFilterAPI returns a new PublicFilterAPI instance.
func NewPublicFilterAPI(backend Backend, lightMode bool) *PublicFilterAPI {
	api := &PublicFilterAPI{
		backend: backend,
		mux:     backend.EventMux(),
		chainDb: backend.ChainDb(),
		events:  NewEventSystem(backend.EventMux(), backend, lightMode),
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
// It is part of the filter package because this filter can be used through the
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

	var (
		rpcSub      = notifier.CreateSubscription()
		matchedLogs = make(chan []*types.Log)
	)

	logsSub, err := api.events.SubscribeLogs(ethereum.FilterQuery(crit), matchedLogs)
	if err != nil {
		return nil, err
	}

	go func() {

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
//
// TODO(karalabe): Kill this in favor of ethereum.FilterQuery.
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
// Default criteria for the from and to block are "latest".
// Using "latest" as block number will return logs for mined blocks.
// Using "pending" as block number returns logs for not yet mined (pending) blocks.
// In case logs are removed (chain reorg) previously returned logs are returned
// again but with the removed property set to true.
//
// In case "fromBlock" > "toBlock" an error is returned.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newfilter
func (api *PublicFilterAPI) NewFilter(crit FilterCriteria) (rpc.ID, error) {
	logs := make(chan []*types.Log)
	logsSub, err := api.events.SubscribeLogs(ethereum.FilterQuery(crit), logs)
	if err != nil {
		return rpc.ID(""), err
	}

	api.filtersMu.Lock()
	api.filters[logsSub.ID] = &filter{typ: LogsSubscription, crit: crit, deadline: time.NewTimer(deadline), logs: make([]*types.Log, 0), s: logsSub}
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

	return logsSub.ID, nil
}

// GetLogs returns logs matching the given argument that are stored within the state.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getlogs
func (api *PublicFilterAPI) GetLogs(ctx context.Context, crit FilterCriteria) ([]*types.Log, error) {
	// Convert the RPC block numbers into internal representations
	if crit.FromBlock == nil {
		crit.FromBlock = big.NewInt(rpc.LatestBlockNumber.Int64())
	}
	if crit.ToBlock == nil {
		crit.ToBlock = big.NewInt(rpc.LatestBlockNumber.Int64())
	}
	// Create and run the filter to get all the logs
	filter := New(api.backend, crit.FromBlock.Int64(), crit.ToBlock.Int64(), crit.Addresses, crit.Topics)

	logs, err := filter.Logs(ctx)
	if err != nil {
		return nil, err
	}
	return returnLogs(logs), err
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
func (api *PublicFilterAPI) GetFilterLogs(ctx context.Context, id rpc.ID) ([]*types.Log, error) {
	api.filtersMu.Lock()
	f, found := api.filters[id]
	api.filtersMu.Unlock()

	if !found || f.typ != LogsSubscription {
		return nil, fmt.Errorf("filter not found")
	}

	begin := rpc.LatestBlockNumber.Int64()
	if f.crit.FromBlock != nil {
		begin = f.crit.FromBlock.Int64()
	}
	end := rpc.LatestBlockNumber.Int64()
	if f.crit.ToBlock != nil {
		end = f.crit.ToBlock.Int64()
	}
	// Create and run the filter to get all the logs
	filter := New(api.backend, begin, end, f.crit.Addresses, f.crit.Topics)

	logs, err := filter.Logs(ctx)
	if err != nil {
		return nil, err
	}
	return returnLogs(logs), nil
}

// GetFilterChanges returns the logs for the filter with the given id since
// last time it was called. This can be used for polling.
//
// For pending transaction and block filters the result is []common.Hash.
// (pending)Log filters return []Log.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getfilterchanges
func (api *PublicFilterAPI) GetFilterChanges(id rpc.ID) (interface{}, error) {
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
			return returnHashes(hashes), nil
		case LogsSubscription:
			logs := f.logs
			f.logs = nil
			return returnLogs(logs), nil
		}
	}

	return []interface{}{}, fmt.Errorf("filter not found")
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
func returnLogs(logs []*types.Log) []*types.Log {
	if logs == nil {
		return []*types.Log{}
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

	if raw.From != nil {
		args.FromBlock = big.NewInt(raw.From.Int64())
	}

	if raw.ToBlock != nil {
		args.ToBlock = big.NewInt(raw.ToBlock.Int64())
	}

	args.Addresses = []common.Address{}

	if raw.Addresses != nil {
		// raw.Address can contain a single address or an array of addresses
		switch rawAddr := raw.Addresses.(type) {
		case []interface{}:
			for i, addr := range rawAddr {
				if strAddr, ok := addr.(string); ok {
					addr, err := decodeAddress(strAddr)
					if err != nil {
						return fmt.Errorf("invalid address at index %d: %v", i, err)
					}
					args.Addresses = append(args.Addresses, addr)
				} else {
					return fmt.Errorf("non-string address at index %d", i)
				}
			}
		case string:
			addr, err := decodeAddress(rawAddr)
			if err != nil {
				return fmt.Errorf("invalid address: %v", err)
			}
			args.Addresses = []common.Address{addr}
		default:
			return errors.New("invalid addresses in query")
		}
	}

	// topics is an array consisting of strings and/or arrays of strings.
	// JSON null values are converted to common.Hash{} and ignored by the filter manager.
	if len(raw.Topics) > 0 {
		args.Topics = make([][]common.Hash, len(raw.Topics))
		for i, t := range raw.Topics {
			switch topic := t.(type) {
			case nil:
				// ignore topic when matching logs

			case string:
				// match specific topic
				top, err := decodeTopic(topic)
				if err != nil {
					return err
				}
				args.Topics[i] = []common.Hash{top}

			case []interface{}:
				// or case e.g. [null, "topic0", "topic1"]
				for _, rawTopic := range topic {
					if rawTopic == nil {
						// null component, match all
						args.Topics[i] = nil
						break
					}
					if topic, ok := rawTopic.(string); ok {
						parsed, err := decodeTopic(topic)
						if err != nil {
							return err
						}
						args.Topics[i] = append(args.Topics[i], parsed)
					} else {
						return fmt.Errorf("invalid topic(s)")
					}
				}
			default:
				return fmt.Errorf("invalid topic(s)")
			}
		}
	}

	return nil
}

func decodeAddress(s string) (common.Address, error) {
	b, err := hexutil.Decode(s)
	if err == nil && len(b) != common.AddressLength {
		err = fmt.Errorf("hex has invalid length %d after decoding", len(b))
	}
	return common.BytesToAddress(b), err
}

func decodeTopic(s string) (common.Hash, error) {
	b, err := hexutil.Decode(s)
	if err == nil && len(b) != common.HashLength {
		err = fmt.Errorf("hex has invalid length %d after decoding", len(b))
	}
	return common.BytesToHash(b), err
}
