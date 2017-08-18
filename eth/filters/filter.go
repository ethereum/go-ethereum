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
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/bitutil"
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
	GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error)
	BloomBitsSections() uint64
	BloomBitsConfig() BloomConfig
	SubscribeTxPreEvent(chan<- core.TxPreEvent) event.Subscription
	SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription
	SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription
	SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription
	GetBloomBits(ctx context.Context, bitIdx uint64, sectionIdxList []uint64) ([][]byte, error)
}

type BloomConfig struct {
	SectionSize    uint64
	MaxRequestLen  int
	MaxRequestWait time.Duration
}

// Filter can be used to retrieve and filter logs.
type Filter struct {
	backend         Backend
	bloomBitsConfig BloomConfig

	db         ethdb.Database
	begin, end int64
	addresses  []common.Address
	topics     [][]common.Hash

	decompress func([]byte, int) ([]byte, error)
	matcher    *bloombits.Matcher
}

// New creates a new filter which uses a bloom filter on blocks to figure out whether
// a particular block is interesting or not.
func New(backend Backend, begin, end int64, addresses []common.Address, topics [][]common.Hash) *Filter {
	return &Filter{
		backend:         backend,
		begin:           begin,
		end:             end,
		addresses:       addresses,
		topics:          topics,
		bloomBitsConfig: backend.BloomBitsConfig(),
		db:              backend.ChainDb(),
		matcher:         bloombits.NewMatcher(backend.BloomBitsConfig().SectionSize, addresses, topics),
		decompress:      bitutil.DecompressBytes,
	}
}

// FindOnce searches the blockchain for matching log entries, returning
// all matching entries from the first block that contains matches,
// updating the start point of the filter accordingly. If no results are
// found, a nil slice is returned.
func (f *Filter) FindOnce(ctx context.Context) ([]*types.Log, error) {
	head, _ := f.backend.HeaderByNumber(ctx, rpc.LatestBlockNumber)
	if head == nil {
		return nil, nil
	}
	headBlockNumber := head.Number.Uint64()

	var beginBlockNo uint64 = uint64(f.begin)
	if f.begin == -1 {
		beginBlockNo = headBlockNumber
	}
	var endBlockNo uint64 = uint64(f.end)
	if f.end == -1 {
		endBlockNo = headBlockNumber
	}

	logs, blockNumber, err := f.getLogs(ctx, beginBlockNo, endBlockNo)
	f.begin = int64(blockNumber + 1)
	return logs, err
}

// Run filters logs with the current parameters set
func (f *Filter) Find(ctx context.Context) (logs []*types.Log, err error) {
	for {
		newLogs, err := f.FindOnce(ctx)
		if len(newLogs) == 0 || err != nil {
			return logs, err
		}
		logs = append(logs, newLogs...)
	}
}

// nextRequest returns the next request to retrieve for the bloombits matcher
func (f *Filter) nextRequest() (bloombits uint, sections []uint64) {
	bloomIndex, ok := f.matcher.AllocSectionQueue()
	if !ok {
		return 0, nil
	}
	if f.bloomBitsConfig.MaxRequestWait > 0 &&
		(f.bloomBitsConfig.MaxRequestLen <= 1 || // SectionCount is always greater than zero after a successful alloc
			f.matcher.SectionCount(bloomIndex) < f.bloomBitsConfig.MaxRequestLen) {
		time.Sleep(f.bloomBitsConfig.MaxRequestWait)
	}
	return bloomIndex, f.matcher.FetchSections(bloomIndex, f.bloomBitsConfig.MaxRequestLen)
}

// serveMatcher serves the bloombits matcher by fetching the requested vectors
// through the filter backend
func (f *Filter) serveMatcher(ctx context.Context, stop chan struct{}, wg *sync.WaitGroup) chan error {
	errChn := make(chan error, 1)
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			defer wg.Done()

			for {
				b, s := f.nextRequest()
				if s == nil {
					return
				}
				data, err := f.backend.GetBloomBits(ctx, uint64(b), s)
				if err != nil {
					select {
					case errChn <- err:
					case <-stop:
					}
					return
				}
				decomp := make([][]byte, len(data))
				for i, d := range data {
					var err error
					if decomp[i], err = f.decompress(d, int(f.bloomBitsConfig.SectionSize/8)); err != nil {
						select {
						case errChn <- err:
						case <-stop:
						}
						return
					}
				}
				f.matcher.Deliver(b, s, decomp)
			}
		}(i)
	}

	return errChn
}

// checkMatches checks if the receipts belonging to the given header contain any log events that
// match the filter criteria. This function is called when the bloom filter signals a potential match.
func (f *Filter) checkMatches(ctx context.Context, header *types.Header) (logs []*types.Log, err error) {
	// Get the logs of the block
	receipts, err := f.backend.GetReceipts(ctx, header.Hash())
	if err != nil {
		return nil, err
	}
	var unfiltered []*types.Log
	for _, receipt := range receipts {
		unfiltered = append(unfiltered, ([]*types.Log)(receipt.Logs)...)
	}
	logs = filterLogs(unfiltered, nil, nil, f.addresses, f.topics)
	if len(logs) > 0 {
		return logs, nil
	}
	return nil, nil
}

func (f *Filter) getLogs(ctx context.Context, start, end uint64) (logs []*types.Log, blockNumber uint64, err error) {
	haveBloomBitsBefore := f.backend.BloomBitsSections() * f.bloomBitsConfig.SectionSize
	if haveBloomBitsBefore > start {
		e := end
		if haveBloomBitsBefore <= e {
			e = haveBloomBitsBefore - 1
		}

		stop := make(chan struct{})
		var wg sync.WaitGroup
		matches := f.matcher.Start(start, e)
		errChn := f.serveMatcher(ctx, stop, &wg)

		defer func() {
			f.matcher.Stop()
			close(stop)
			wg.Wait()
		}()

	loop:
		for {
			select {
			case i, ok := <-matches:
				if !ok {
					break loop
				}

				blockNumber := rpc.BlockNumber(i)
				header, err := f.backend.HeaderByNumber(ctx, blockNumber)
				if header == nil || err != nil {
					return logs, end, err
				}

				logs, err := f.checkMatches(ctx, header)
				if err != nil {
					return nil, end, err
				}
				if logs != nil {
					return logs, i, nil
				}
			case err := <-errChn:
				return logs, end, err
			case <-ctx.Done():
				return nil, end, ctx.Err()
			}
		}

		if end < haveBloomBitsBefore {
			return logs, end, nil
		}
		start = haveBloomBitsBefore
	}

	// search the rest with regular block-by-block bloom filtering
	for i := start; i <= end; i++ {
		blockNumber := rpc.BlockNumber(i)
		header, err := f.backend.HeaderByNumber(ctx, blockNumber)
		if header == nil || err != nil {
			return logs, end, err
		}

		// Use bloom filtering to see if this block is interesting given the
		// current parameters
		if f.bloomFilter(header.Bloom) {
			logs, err := f.checkMatches(ctx, header)
			if err != nil {
				return nil, end, err
			}
			if logs != nil {
				return logs, i, nil
			}
		}
	}

	return logs, end, nil
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

		logTopics := make([]common.Hash, len(topics))
		copy(logTopics, log.Topics)

		// If the to filtered topics is greater than the amount of topics in logs, skip.
		if len(topics) > len(log.Topics) {
			continue Logs
		}

		for i, topics := range topics {
			var match bool
			for _, topic := range topics {
				// common.Hash{} is a match all (wildcard)
				if (topic == common.Hash{}) || log.Topics[i] == topic {
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

func (f *Filter) bloomFilter(bloom types.Bloom) bool {
	return bloomFilter(bloom, f.addresses, f.topics)
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
		var included bool
		for _, topic := range sub {
			if (topic == common.Hash{}) || types.BloomLookup(bloom, topic) {
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
