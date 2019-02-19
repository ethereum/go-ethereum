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

package filters

import (
	"context"
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/bloombits"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
)

type Backend interface {
	ChainDb() ethdb.Database
	EventMux() *event.TypeMux
	HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error)
	HeaderByHash(ctx context.Context, blockHash common.Hash) (*types.Header, error)
	GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error)
	GetLogs(ctx context.Context, blockHash common.Hash) ([][]*types.Log, error)

	SubscribeNewTxsEvent(chan<- core.NewTxsEvent) event.Subscription
	SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription
	SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription
	SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription

	BloomStatus() (uint64, uint64)
	ServiceFilter(ctx context.Context, session *bloombits.MatcherSession)
	FilterTaskChannel() chan *BlockFilterTask
}

// BlockFilterTask represents a request for log filtering task assignments for a given
// block number, or a response for such a request.
// It can also have the actual results set to be used as a delivery data struct.
//
// The contest and error fields are used by the light client to terminate matching
// early if an error is enountered on some path of the pipeline.
type BlockFilterTask struct {
	blockNumber uint64
	checkBloom  bool
	filter      *Filter
	context     context.Context

	logs []*types.Log
	err  error
	done chan struct{}
}

// Do is a callback function called for each block filter task by ServeFilterTasks
func (t *BlockFilterTask) Do() {
	defer close(t.done)

	header, err := t.filter.backend.HeaderByNumber(t.context, rpc.BlockNumber(t.blockNumber))
	if err != nil {
		t.err = err
		return
	}
	if header == nil {
		t.err = errors.New("Header not found")
		return
	}
	if !t.checkBloom || bloomFilter(header.Bloom, t.filter.addresses, t.filter.topics) {
		t.logs, t.err = t.filter.checkMatches(t.context, header)
	}
}

// filterTaskServiceThreads is the number of goroutines used globally by a filter
// backend to service block filter tasks for all running filters.
const filterTaskServiceThreads = 16

// ServeFilterTasks is called by the filter backend to start up filter task service threads
func ServeFilterTasks(tasks chan *BlockFilterTask, stop chan bool) {
	for i := 0; i < filterTaskServiceThreads; i++ {
		go func() {
			for {
				select {
				case <-stop:
					return

				case task := <-tasks:
					task.Do()
				}
			}
		}()
	}
}

// Filter can be used to retrieve and filter logs.
type Filter struct {
	backend Backend

	db        ethdb.Database
	addresses []common.Address
	topics    [][]common.Hash

	block      common.Hash // Block hash if filtering a single block
	begin, end int64       // Range interval if filtering multiple blocks

	matcher *bloombits.Matcher
}

// NewRangeFilter creates a new filter which uses a bloom filter on blocks to
// figure out whether a particular block is interesting or not.
func NewRangeFilter(backend Backend, begin, end int64, addresses []common.Address, topics [][]common.Hash) *Filter {
	// Flatten the address and topic filter clauses into a single bloombits filter
	// system. Since the bloombits are not positional, nil topics are permitted,
	// which get flattened into a nil byte slice.
	var filters [][][]byte
	if len(addresses) > 0 {
		filter := make([][]byte, len(addresses))
		for i, address := range addresses {
			filter[i] = address.Bytes()
		}
		filters = append(filters, filter)
	}
	for _, topicList := range topics {
		filter := make([][]byte, len(topicList))
		for i, topic := range topicList {
			filter[i] = topic.Bytes()
		}
		filters = append(filters, filter)
	}
	size, _ := backend.BloomStatus()

	// Create a generic filter and convert it into a range filter
	filter := newFilter(backend, addresses, topics)

	filter.matcher = bloombits.NewMatcher(size, filters)
	filter.begin = begin
	filter.end = end

	return filter
}

// NewBlockFilter creates a new filter which directly inspects the contents of
// a block to figure out whether it is interesting or not.
func NewBlockFilter(backend Backend, block common.Hash, addresses []common.Address, topics [][]common.Hash) *Filter {
	// Create a generic filter and convert it into a block filter
	filter := newFilter(backend, addresses, topics)
	filter.block = block
	return filter
}

// newFilter creates a generic filter that can either filter based on a block hash,
// or based on range queries. The search criteria needs to be explicitly set.
func newFilter(backend Backend, addresses []common.Address, topics [][]common.Hash) *Filter {
	return &Filter{
		backend:   backend,
		addresses: addresses,
		topics:    topics,
		db:        backend.ChainDb(),
	}
}

// Logs searches the blockchain for matching log entries, returning all from the
// first block that contains matches, updating the start of the filter accordingly.
func (f *Filter) Logs(ctx context.Context) ([]*types.Log, error) {
	// If we're doing singleton block filtering, execute and return
	if f.block != (common.Hash{}) {
		header, err := f.backend.HeaderByHash(ctx, f.block)
		if err != nil {
			return nil, err
		}
		if header == nil {
			return nil, errors.New("unknown block")
		}
		return f.blockLogs(ctx, header)
	}
	// Figure out the limits of the filter range
	header, _ := f.backend.HeaderByNumber(ctx, rpc.LatestBlockNumber)
	if header == nil {
		return nil, nil
	}
	head := header.Number.Uint64()

	if f.begin == -1 {
		f.begin = int64(head)
	}
	end := uint64(f.end)
	if f.end == -1 {
		end = head
	}
	// Gather all indexed logs, and finish with non indexed ones
	var (
		logs []*types.Log
		err  error
		wg   sync.WaitGroup
	)
	tasks := make(chan *BlockFilterTask, 64)
	wg.Add(1)
	go func() {
		for task := range tasks {
			if err == nil {
				select {
				case <-task.done:
				case <-ctx.Done():
					err = ctx.Err()
					continue
				}
				if len(task.logs) != 0 {
					logs = append(logs, task.logs...)
				}
				err = task.err
			}
		}
		wg.Done()
	}()

	size, sections := f.backend.BloomStatus()
	if indexed := sections * size; indexed > uint64(f.begin) {
		if indexed > end {
			f.indexedLogs(ctx, tasks, end)
		} else {
			f.indexedLogs(ctx, tasks, indexed-1)
		}
		if err != nil {
			return logs, err
		}
	}
	f.unindexedLogs(ctx, tasks, end)
	close(tasks)
	wg.Wait()
	return logs, err
}

// indexedLogs returns the logs matching the filter criteria based on the bloom
// bits indexed available locally or via the network.
func (f *Filter) indexedLogs(ctx context.Context, tasks chan *BlockFilterTask, end uint64) error {
	// Create a matcher session and request servicing from the backend
	matches := make(chan uint64, 64)

	session, err := f.matcher.Start(ctx, uint64(f.begin), end, matches)
	if err != nil {
		return err
	}
	defer session.Close()

	f.backend.ServiceFilter(ctx, session)

	// Iterate over the matches until exhausted or context closed
	for {
		select {
		case number, ok := <-matches:
			// Abort if all matches have been fulfilled
			if !ok {
				err := session.Error()
				if err == nil {
					f.begin = int64(end) + 1
				}
				return err
			}
			f.begin = int64(number) + 1

			task := &BlockFilterTask{
				blockNumber: number,
				checkBloom:  false,
				filter:      f,
				context:     ctx,
				done:        make(chan struct{}),
			}
			select {
			case f.backend.FilterTaskChannel() <- task:
			case <-ctx.Done():
				return ctx.Err()
			}
			select {
			case tasks <- task:
			case <-ctx.Done():
				return ctx.Err()
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// indexedLogs returns the logs matching the filter criteria based on raw block
// iteration and bloom matching.
func (f *Filter) unindexedLogs(ctx context.Context, tasks chan *BlockFilterTask, end uint64) error {
	for ; f.begin <= int64(end); f.begin++ {
		task := &BlockFilterTask{
			blockNumber: uint64(f.begin),
			checkBloom:  true,
			filter:      f,
			context:     ctx,
			done:        make(chan struct{}),
		}
		select {
		case f.backend.FilterTaskChannel() <- task:
		case <-ctx.Done():
			return ctx.Err()
		}
		select {
		case tasks <- task:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// blockLogs returns the logs matching the filter criteria within a single block.
func (f *Filter) blockLogs(ctx context.Context, header *types.Header) (logs []*types.Log, err error) {
	if bloomFilter(header.Bloom, f.addresses, f.topics) {
		found, err := f.checkMatches(ctx, header)
		if err != nil {
			return logs, err
		}
		logs = found
	}
	return logs, nil
}

// checkMatches checks if the receipts belonging to the given header contain any log events that
// match the filter criteria. This function is called when the bloom filter signals a potential match.
func (f *Filter) checkMatches(ctx context.Context, header *types.Header) (logs []*types.Log, err error) {
	// Get the logs of the block
	logsList, err := f.backend.GetLogs(ctx, header.Hash())
	if err != nil {
		return nil, err
	}
	var unfiltered []*types.Log
	for _, logs := range logsList {
		unfiltered = append(unfiltered, logs...)
	}
	logs = filterLogs(unfiltered, nil, nil, f.addresses, f.topics)
	if len(logs) > 0 {
		// We have matching logs, check if we need to resolve full logs via the light client
		if logs[0].TxHash == (common.Hash{}) {
			receipts, err := f.backend.GetReceipts(ctx, header.Hash())
			if err != nil {
				return nil, err
			}
			unfiltered = unfiltered[:0]
			for _, receipt := range receipts {
				unfiltered = append(unfiltered, receipt.Logs...)
			}
			logs = filterLogs(unfiltered, nil, nil, f.addresses, f.topics)
		}
		return logs, nil
	}
	return nil, nil
}

func includes(addresses []common.Address, a common.Address) bool {
	for _, addr := range addresses {
		if addr == a {
			return true
		}
	}

	return false
}

// filterLogs creates a slice of logs matching the given criteria.
func filterLogs(logs []*types.Log, fromBlock, toBlock *big.Int, addresses []common.Address, topics [][]common.Hash) []*types.Log {
	var ret []*types.Log
Logs:
	for _, log := range logs {
		if fromBlock != nil && fromBlock.Int64() >= 0 && fromBlock.Uint64() > log.BlockNumber {
			continue
		}
		if toBlock != nil && toBlock.Int64() >= 0 && toBlock.Uint64() < log.BlockNumber {
			continue
		}

		if len(addresses) > 0 && !includes(addresses, log.Address) {
			continue
		}
		// If the to filtered topics is greater than the amount of topics in logs, skip.
		if len(topics) > len(log.Topics) {
			continue Logs
		}
		for i, sub := range topics {
			match := len(sub) == 0 // empty rule set == wildcard
			for _, topic := range sub {
				if log.Topics[i] == topic {
					match = true
					break
				}
			}
			if !match {
				continue Logs
			}
		}
		ret = append(ret, log)
	}
	return ret
}

func bloomFilter(bloom types.Bloom, addresses []common.Address, topics [][]common.Hash) bool {
	if len(addresses) > 0 {
		var included bool
		for _, addr := range addresses {
			if types.BloomLookup(bloom, addr) {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}

	for _, sub := range topics {
		included := len(sub) == 0 // empty rule set == wildcard
		for _, topic := range sub {
			if types.BloomLookup(bloom, topic) {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}
	return true
}
