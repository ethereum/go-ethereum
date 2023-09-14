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

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/history"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	logger "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	errInvalidTopic       = errors.New("invalid topic(s)")
	errFilterNotFound     = errors.New("filter not found")
	errConnectDropped     = errors.New("connection dropped")
	errInvalidToBlock     = errors.New("log subscription does not support history block range")
	errInvalidFromBlock   = errors.New("from block can be only a number, or \"safe\", or \"finalized\"")
	errClientUnsubscribed = errors.New("client unsubscribed")
)

// filter is a helper struct that holds meta information over the filter type
// and associated subscription in the event system.
type filter struct {
	typ      Type
	deadline *time.Timer // filter is inactive when deadline triggers
	hashes   []common.Hash
	fullTx   bool
	txs      []*types.Transaction
	crit     FilterCriteria
	logs     []*types.Log
	s        *Subscription // associated subscription in event system
}

// FilterAPI offers support to create and manage filters. This will allow external clients to retrieve various
// information related to the Ethereum protocol such as blocks, transactions and logs.
type FilterAPI struct {
	sys       *FilterSystem
	events    *EventSystem
	filtersMu sync.Mutex
	filters   map[rpc.ID]*filter
	timeout   time.Duration
}

// NewFilterAPI returns a new FilterAPI instance.
func NewFilterAPI(system *FilterSystem) *FilterAPI {
	api := &FilterAPI{
		sys:     system,
		events:  NewEventSystem(system),
		filters: make(map[rpc.ID]*filter),
		timeout: system.cfg.Timeout,
	}
	go api.timeoutLoop(system.cfg.Timeout)

	return api
}

// timeoutLoop runs at the interval set by 'timeout' and deletes filters
// that have not been recently used. It is started when the API is created.
func (api *FilterAPI) timeoutLoop(timeout time.Duration) {
	var toUninstall []*Subscription
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
		case <-api.events.chainSub.Err():
			return
		}
		api.filtersMu.Lock()
		for id, f := range api.filters {
			select {
			case <-f.deadline.C:
				toUninstall = append(toUninstall, f.s)
				delete(api.filters, id)
			default:
				continue
			}
		}
		api.filtersMu.Unlock()

		// Unsubscribes are processed outside the lock to avoid the following scenario:
		// event loop attempts broadcasting events to still active filters while
		// Unsubscribe is waiting for it to process the uninstall request.
		for _, s := range toUninstall {
			s.Unsubscribe()
		}
		toUninstall = nil
	}
}

// NewPendingTransactionFilter creates a filter that fetches pending transactions
// as transactions enter the pending state.
//
// It is part of the filter package because this filter can be used through the
// `eth_getFilterChanges` polling method that is also used for log filters.
func (api *FilterAPI) NewPendingTransactionFilter(fullTx *bool) rpc.ID {
	var (
		pendingTxs   = make(chan []*types.Transaction)
		pendingTxSub = api.events.SubscribePendingTxs(pendingTxs)
	)

	api.filtersMu.Lock()
	api.filters[pendingTxSub.ID] = &filter{typ: PendingTransactionsSubscription, fullTx: fullTx != nil && *fullTx, deadline: time.NewTimer(api.timeout), txs: make([]*types.Transaction, 0), s: pendingTxSub}
	api.filtersMu.Unlock()

	go func() {
		for {
			select {
			case pTx := <-pendingTxs:
				api.filtersMu.Lock()
				if f, found := api.filters[pendingTxSub.ID]; found {
					f.txs = append(f.txs, pTx...)
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

// NewPendingTransactions creates a subscription that is triggered each time a
// transaction enters the transaction pool. If fullTx is true the full tx is
// sent to the client, otherwise the hash is sent.
func (api *FilterAPI) NewPendingTransactions(ctx context.Context, fullTx *bool) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		txs := make(chan []*types.Transaction, 128)
		pendingTxSub := api.events.SubscribePendingTxs(txs)
		defer pendingTxSub.Unsubscribe()

		chainConfig := api.sys.backend.ChainConfig()

		for {
			select {
			case txs := <-txs:
				// To keep the original behaviour, send a single tx hash in one notification.
				// TODO(rjl493456442) Send a batch of tx hashes in one notification
				latest := api.sys.backend.CurrentHeader()
				for _, tx := range txs {
					if fullTx != nil && *fullTx {
						rpcTx := ethapi.NewRPCPendingTransaction(tx, latest, chainConfig)
						notifier.Notify(rpcSub.ID, rpcTx)
					} else {
						notifier.Notify(rpcSub.ID, tx.Hash())
					}
				}
			case <-rpcSub.Err():
				return
			}
		}
	}()

	return rpcSub, nil
}

// NewBlockFilter creates a filter that fetches blocks that are imported into the chain.
// It is part of the filter package since polling goes with eth_getFilterChanges.
func (api *FilterAPI) NewBlockFilter() rpc.ID {
	var (
		headers   = make(chan *types.Header)
		headerSub = api.events.SubscribeNewHeads(headers)
	)

	api.filtersMu.Lock()
	api.filters[headerSub.ID] = &filter{typ: BlocksSubscription, deadline: time.NewTimer(api.timeout), hashes: make([]common.Hash, 0), s: headerSub}
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
func (api *FilterAPI) NewHeads(ctx context.Context) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		headers := make(chan *types.Header)
		headersSub := api.events.SubscribeNewHeads(headers)
		defer headersSub.Unsubscribe()

		for {
			select {
			case h := <-headers:
				notifier.Notify(rpcSub.ID, h)
			case <-rpcSub.Err():
				return
			}
		}
	}()

	return rpcSub, nil
}

// notifier is used for broadcasting data(eg: logs) to rpc receivers
// used in unit testing.
type notifier interface {
	Notify(id rpc.ID, data interface{}) error
	Closed() <-chan interface{}
}

// Logs creates a subscription that fires for all historical
// and new logs that match the given filter criteria.
func (api *FilterAPI) Logs(ctx context.Context, crit FilterCriteria) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}
	rpcSub := notifier.CreateSubscription()
	err := api.logs(ctx, notifier, rpcSub, crit)
	return rpcSub, err
}

// logs is the inner implementation of logs subscription.
// The following criteria are valid:
// * from: nil, to: nil -> yield live logs.
// * from: blockNum | safe | finalized, to: nil -> historical beginning at `from` to head, then live logs.
// * Every other case should fail with an error.
func (api *FilterAPI) logs(ctx context.Context, notifier notifier, rpcSub *rpc.Subscription, crit FilterCriteria) error {
	if crit.ToBlock != nil {
		return errInvalidToBlock
	}
	if crit.FromBlock == nil {
		return api.liveLogs(notifier, rpcSub, crit)
	}
	from := rpc.BlockNumber(crit.FromBlock.Int64())
	switch from {
	case rpc.LatestBlockNumber, rpc.PendingBlockNumber:
		return errInvalidFromBlock
	case rpc.SafeBlockNumber, rpc.FinalizedBlockNumber:
		header, err := api.sys.backend.HeaderByNumber(ctx, from)
		if err != nil {
			return err
		}
		from = rpc.BlockNumber(header.Number.Int64())
	}
	if from < 0 {
		return errInvalidFromBlock
	}
	return api.histLogs(notifier, rpcSub, int64(from), crit)
}

// liveLogs only retrieves live logs.
func (api *FilterAPI) liveLogs(notifier notifier, rpcSub *rpc.Subscription, crit FilterCriteria) error {
	matchedLogs := make(chan []*types.Log)
	logsSub, err := api.events.SubscribeLogs(ethereum.FilterQuery(crit), matchedLogs)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case logs := <-matchedLogs:
				notifyLogsIf(notifier, rpcSub.ID, logs, nil)
			case <-rpcSub.Err(): // client send an unsubscribe request
				logsSub.Unsubscribe()
				return
			case <-notifier.Closed(): // connection dropped
				logsSub.Unsubscribe()
				return
			}
		}
	}()
	return nil
}

// histLogs retrieves logs older than current header.
func (api *FilterAPI) histLogs(notifier notifier, rpcSub *rpc.Subscription, from int64, crit FilterCriteria) error {
	// Subscribe the Live logs
	// if an ChainReorg occurred,
	// we will first recv the old chain's deleted logs in descending order,
	// and then the new chain's added logs in descending order
	// see core/blockchain.go#reorg(oldHead *types.Header, newHead *types.Block) for more details
	// if an reorg happened between `from` and `to`, we will need to think about some scenarios:
	// 1. if a reorg occurs after the currently delivered block, then because this is happening in the future, has nothing to do with the current historical sync, we can just ignore it.
	// 2. if a reorg occurs before the currently delivered block, then we need to stop the historical delivery, and send all replaced logs instead
	var (
		liveLogs = make(chan []*types.Log)
		histLogs = make(chan []*types.Log)
		histDone = make(chan error)
	)
	liveLogsSub, err := api.events.SubscribeLogs(ethereum.FilterQuery(crit), liveLogs)
	if err != nil {
		return err
	}

	// The original request ctx will be canceled as soon as the parent goroutine
	// returns a subscription. Use a new context instead.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		histDone <- api.doHistLogs(ctx, from, crit.Addresses, crit.Topics, histLogs)
	}()

	// Compose and notify the logs from liveLogs and histLogs
	go func() {
		defer func() {
			liveLogsSub.Unsubscribe()
			cancel()
		}()
		var (
			delivered uint64
			liveOnly  bool
			reorged   bool
			staleHash common.Hash
		)
		for {
			select {
			case err := <-histDone:
				if err != nil {
					logger.Warn("History logs delivery failed", "err", err)
					return
				}
				// Else historical logs are all delivered, let's switch to live mode
				logger.Info("History logs delivery finished, and now enter into live mode", "delivered", delivered)
				liveOnly = true
				histLogs = nil

			case logs := <-liveLogs:
				if len(logs) == 0 {
					continue
				}
				if liveOnly {
					notifyLogsIf(notifier, rpcSub.ID, logs, nil)
					continue
				}
				reorgBlock := logs[0].BlockNumber
				if !reorged && reorgBlock <= delivered {
					logger.Info("Reorg detected", "reorgBlock", reorgBlock, "delivered", delivered)
					reorged = true
				}
				if !reorged {
					// Reorg in future. Remember fork point.
					if logs[0].Removed && staleHash == (common.Hash{}) {
						staleHash = logs[0].BlockHash
					}
					continue
				}
				if logs[0].Removed {
					// Send removed logs notification up until the point
					// we have delivered logs from old chain.
					notifyLogsIf(notifier, rpcSub.ID, logs, func(log *types.Log) bool {
						return log.BlockNumber <= delivered
					})
				} else {
					// New logs are emitted for the whole new chain since the reorg block.
					// Send them all.
					notifyLogsIf(notifier, rpcSub.ID, logs, nil)
				}

			case logs := <-histLogs:
				if len(logs) == 0 {
					continue
				}
				if reorged {
					continue
				}
				if logs[0].BlockHash == staleHash {
					// We have reached the fork point and the historical producer
					// is emitting old logs because of delay. Restart the process
					// from last delivered block.
					logger.Info("Restarting historical logs delivery", "from", logs[0].BlockNumber, "delivered", delivered)
					liveLogsSub.Unsubscribe()
					// Stop hist logs fetcher
					cancel()
					if err := api.histLogs(notifier, rpcSub, int64(logs[0].BlockNumber), crit); err != nil {
						logger.Warn("failed to restart historical logs delivery", "err", err)
					}
					return
				}
				notifyLogsIf(notifier, rpcSub.ID, logs, nil)
				// Assuming batch = all logs of a single block
				delivered = logs[0].BlockNumber

			case <-rpcSub.Err(): // client send an unsubscribe request
				return
			case <-notifier.Closed(): // connection dropped
				return
			}
		}
	}()

	return nil
}

// doHistLogs retrieves the logs older than current header, and forward them to the histLogs channel.
func (api *FilterAPI) doHistLogs(ctx context.Context, from int64, addrs []common.Address, topics [][]common.Hash, histLogs chan<- []*types.Log) error {
	// Fetch logs from a range of blocks.
	fetchRange := func(from, to int64) error {
		f := api.sys.NewRangeFilter(from, to, addrs, topics)
		logsCh, errChan := f.rangeLogsAsync(ctx)
		for {
			select {
			case logs := <-logsCh:
				select {
				case histLogs <- logs:
				case <-ctx.Done():
					// Flush out all logs until the range filter voluntarily exits.
					continue
				}
			case err := <-errChan:
				return err
			}
		}
	}

	for {
		// Get the latest block header.
		header := api.sys.backend.CurrentHeader()
		if header == nil {
			return errors.New("unexpected error: no header block found")
		}
		head := header.Number.Int64()
		if from > head {
			logger.Info("Finish historical sync", "from", from, "head", head)
			return nil
		}
		if err := fetchRange(from, head); err != nil {
			if errors.Is(err, context.Canceled) {
				logger.Info("Historical logs delivery canceled", "from", from, "to", head)
				return nil
			}
			return err
		}
		// Move forward to the next batch
		from = head + 1
	}
}

func notifyLogsIf(notifier notifier, id rpc.ID, logs []*types.Log, cond func(log *types.Log) bool) {
	for _, log := range logs {
		if cond == nil || cond(log) {
			log := log
			notifier.Notify(id, &log)
		}
	}
}

// FilterCriteria represents a request to create a new filter.
// Same as ethereum.FilterQuery but with UnmarshalJSON() method.
type FilterCriteria ethereum.FilterQuery

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
func (api *FilterAPI) NewFilter(crit FilterCriteria) (rpc.ID, error) {
	logs := make(chan []*types.Log)
	logsSub, err := api.events.SubscribeLogs(ethereum.FilterQuery(crit), logs)
	if err != nil {
		return "", err
	}

	api.filtersMu.Lock()
	api.filters[logsSub.ID] = &filter{typ: LogsSubscription, crit: crit, deadline: time.NewTimer(api.timeout), logs: make([]*types.Log, 0), s: logsSub}
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
func (api *FilterAPI) GetLogs(ctx context.Context, crit FilterCriteria) ([]*types.Log, error) {
	if len(crit.Topics) > maxTopics {
		return nil, errExceedMaxTopics
	}
	if len(crit.Addresses) > maxAddresses {
		return nil, errExceedMaxAddresses
	}

	var filter *Filter
	if crit.BlockHash != nil {
		if crit.FromBlock != nil || crit.ToBlock != nil {
			return nil, errBlockHashWithRange
		}

		// Block filter requested, construct a single-shot filter
		filter = api.sys.NewBlockFilter(*crit.BlockHash, crit.Addresses, crit.Topics)
	} else {
		// Convert the RPC block numbers into internal representations
		begin := rpc.LatestBlockNumber.Int64()
		if crit.FromBlock != nil {
			begin = crit.FromBlock.Int64()
		}
		end := rpc.LatestBlockNumber.Int64()
		if crit.ToBlock != nil {
			end = crit.ToBlock.Int64()
		}
		// Block numbers below 0 are special cases.
		if begin > 0 && end > 0 && begin > end {
			return nil, errInvalidBlockRange
		}
		if begin >= 0 && begin < int64(api.events.backend.HistoryPruningCutoff()) {
			return nil, &history.PrunedHistoryError{}
		}
		// Construct the range filter
		filter = api.sys.NewRangeFilter(begin, end, crit.Addresses, crit.Topics)
	}

	// Run the filter and return all the logs
	logs, err := filter.Logs(ctx)
	if err != nil {
		return nil, err
	}
	return returnLogs(logs), err
}

// UninstallFilter removes the filter with the given filter id.
func (api *FilterAPI) UninstallFilter(id rpc.ID) bool {
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
func (api *FilterAPI) GetFilterLogs(ctx context.Context, id rpc.ID) ([]*types.Log, error) {
	api.filtersMu.Lock()
	f, found := api.filters[id]
	api.filtersMu.Unlock()

	if !found || f.typ != LogsSubscription {
		return nil, errFilterNotFound
	}

	var filter *Filter
	if f.crit.BlockHash != nil {
		// Block filter requested, construct a single-shot filter
		filter = api.sys.NewBlockFilter(*f.crit.BlockHash, f.crit.Addresses, f.crit.Topics)
	} else {
		// Convert the RPC block numbers into internal representations
		begin := rpc.LatestBlockNumber.Int64()
		if f.crit.FromBlock != nil {
			begin = f.crit.FromBlock.Int64()
		}
		end := rpc.LatestBlockNumber.Int64()
		if f.crit.ToBlock != nil {
			end = f.crit.ToBlock.Int64()
		}
		// Construct the range filter
		filter = api.sys.NewRangeFilter(begin, end, f.crit.Addresses, f.crit.Topics)
	}
	// Run the filter and return all the logs
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
func (api *FilterAPI) GetFilterChanges(id rpc.ID) (interface{}, error) {
	api.filtersMu.Lock()
	defer api.filtersMu.Unlock()

	chainConfig := api.sys.backend.ChainConfig()
	latest := api.sys.backend.CurrentHeader()

	if f, found := api.filters[id]; found {
		if !f.deadline.Stop() {
			// timer expired but filter is not yet removed in timeout loop
			// receive timer value and reset timer
			<-f.deadline.C
		}
		f.deadline.Reset(api.timeout)

		switch f.typ {
		case BlocksSubscription:
			hashes := f.hashes
			f.hashes = nil
			return returnHashes(hashes), nil
		case PendingTransactionsSubscription:
			if f.fullTx {
				txs := make([]*ethapi.RPCTransaction, 0, len(f.txs))
				for _, tx := range f.txs {
					txs = append(txs, ethapi.NewRPCPendingTransaction(tx, latest, chainConfig))
				}
				f.txs = nil
				return txs, nil
			} else {
				hashes := make([]common.Hash, 0, len(f.txs))
				for _, tx := range f.txs {
					hashes = append(hashes, tx.Hash())
				}
				f.txs = nil
				return hashes, nil
			}
		case LogsSubscription:
			logs := f.logs
			f.logs = nil
			return returnLogs(logs), nil
		}
	}

	return []interface{}{}, errFilterNotFound
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
		BlockHash *common.Hash     `json:"blockHash"`
		FromBlock *rpc.BlockNumber `json:"fromBlock"`
		ToBlock   *rpc.BlockNumber `json:"toBlock"`
		Addresses interface{}      `json:"address"`
		Topics    []interface{}    `json:"topics"`
	}

	var raw input
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if raw.BlockHash != nil {
		if raw.FromBlock != nil || raw.ToBlock != nil {
			// BlockHash is mutually exclusive with FromBlock/ToBlock criteria
			return errors.New("cannot specify both BlockHash and FromBlock/ToBlock, choose one or the other")
		}
		args.BlockHash = raw.BlockHash
	} else {
		if raw.FromBlock != nil {
			args.FromBlock = big.NewInt(raw.FromBlock.Int64())
		}

		if raw.ToBlock != nil {
			args.ToBlock = big.NewInt(raw.ToBlock.Int64())
		}
	}

	args.Addresses = []common.Address{}

	if raw.Addresses != nil {
		// raw.Address can contain a single address or an array of addresses
		switch rawAddr := raw.Addresses.(type) {
		case []interface{}:
			if len(rawAddr) > maxAddresses {
				return errExceedMaxAddresses
			}
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
	if len(raw.Topics) > maxTopics {
		return errExceedMaxTopics
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
				if len(topic) > maxSubTopics {
					return errExceedMaxTopics
				}
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
						return errInvalidTopic
					}
				}
			default:
				return errInvalidTopic
			}
		}
	}

	return nil
}

func decodeAddress(s string) (common.Address, error) {
	b, err := hexutil.Decode(s)
	if err == nil && len(b) != common.AddressLength {
		err = fmt.Errorf("hex has invalid length %d after decoding; expected %d for address", len(b), common.AddressLength)
	}
	return common.BytesToAddress(b), err
}

func decodeTopic(s string) (common.Hash, error) {
	b, err := hexutil.Decode(s)
	if err == nil && len(b) != common.HashLength {
		err = fmt.Errorf("hex has invalid length %d after decoding; expected %d for topic", len(b), common.HashLength)
	}
	return common.BytesToHash(b), err
}
