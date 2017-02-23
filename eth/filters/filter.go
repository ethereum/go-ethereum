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
	"math"
	"time"

	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/bloombits"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/net/context"
)

type Backend interface {
	ChainDb() ethdb.Database
	EventMux() *event.TypeMux
	HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error)
	GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error)
	GetBloomBits(ctx context.Context, bitIdx uint64, sectionIdxList []uint64) ([]bloombits.CompVector, error)
}

// Filter can be used to retrieve and filter logs.
type Filter struct {
	backend                 Backend
	useMipMap, useBloomBits bool

	created time.Time

	db         ethdb.Database
	begin, end int64
	addresses  []common.Address
	topics     [][]common.Hash

	matcher *bloombits.Matcher
}

// New creates a new filter which uses a bloom filter on blocks to figure out whether
// a particular block is interesting or not.
// MipMaps allow past blocks to be searched much more efficiently, but are not available
// to light clients.
func New(backend Backend, useMipMap bool) *Filter {
	return &Filter{
		backend:      backend,
		useMipMap:    useMipMap,
		useBloomBits: !useMipMap,
		db:           backend.ChainDb(),
		matcher:      bloombits.NewMatcher(),
	}
}

// SetBeginBlock sets the earliest block for filtering.
// -1 = latest block (i.e., the current block)
// hash = particular hash from-to
func (f *Filter) SetBeginBlock(begin int64) {
	f.begin = begin
}

// SetEndBlock sets the latest block for filtering.
func (f *Filter) SetEndBlock(end int64) {
	f.end = end
}

// SetAddresses matches only logs that are generated from addresses that are included
// in the given addresses.
func (f *Filter) SetAddresses(addr []common.Address) {
	f.addresses = addr
	if f.useBloomBits {
		f.matcher.SetAddresses(addr)
	}
}

// SetTopics matches only logs that have topics matching the given topics.
func (f *Filter) SetTopics(topics [][]common.Hash) {
	f.topics = topics
	if f.useBloomBits {
		f.matcher.SetTopics(topics)
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

	// if no addresses are present we can't make use of fast search which
	// uses the mipmap bloom filters to check for fast inclusion and uses
	// higher range probability in order to ensure at least a false positive
	if !f.useMipMap || len(f.addresses) == 0 {
		logs, blockNumber, err := f.getLogs(ctx, beginBlockNo, endBlockNo)
		f.begin = int64(blockNumber + 1)
		return logs, err
	}

	logs, blockNumber := f.mipFind(beginBlockNo, endBlockNo, 0)
	f.begin = int64(blockNumber + 1)
	return logs, nil
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

func (f *Filter) mipFind(start, end uint64, depth int) (logs []*types.Log, blockNumber uint64) {
	level := core.MIPMapLevels[depth]
	// normalise numerator so we can work in level specific batches and
	// work with the proper range checks
	for num := start / level * level; num <= end; num += level {
		// find addresses in bloom filters
		bloom := core.GetMipmapBloom(f.db, num, level)
		// Don't bother checking the first time through the loop - we're probably picking
		// up where a previous run left off.
		first := true
		for _, addr := range f.addresses {
			if first || bloom.TestBytes(addr[:]) {
				first = false
				// range check normalised values and make sure that
				// we're resolving the correct range instead of the
				// normalised values.
				start := uint64(math.Max(float64(num), float64(start)))
				end := uint64(math.Min(float64(num+level-1), float64(end)))
				if depth+1 == len(core.MIPMapLevels) {
					l, blockNumber, _ := f.getLogs(context.Background(), start, end)
					if len(l) > 0 {
						return l, blockNumber
					}
				} else {
					l, blockNumber := f.mipFind(start, end, depth+1)
					if len(l) > 0 {
						return l, blockNumber
					}
				}
			}
		}
	}

	return nil, end
}

func (f *Filter) serveMatcher(ctx context.Context, stop chan struct{}) chan error {
	errChn := make(chan error)
	for i := 0; i < 10; i++ {
		go func(i int) {
			for {
				//fmt.Println(i, "NextRequest")
				b, s := f.matcher.NextRequest(stop)
				//fmt.Println(i, "NextRequest ret", b, s)
				if s == nil {
					return
				}
				data, err := f.backend.GetBloomBits(ctx, uint64(b), s)
				//fmt.Println(i, "GetBloomBits", len(data), err)
				if err != nil {
					f.matcher.Deliver(b, s, nil)
					errChn <- err
					return
				}
				decomp := make([]bloombits.BitVector, len(data))
				for i, d := range data {
					decomp[i] = bloombits.DecompressBloomBits(bloombits.CompVector(d))
				}
				//fmt.Println(i, "Deliver")
				f.matcher.Deliver(b, s, decomp)
				//fmt.Println(i, "Deliver ret")
			}
		}(i)
	}

	return errChn
}

func (f *Filter) getLogs(ctx context.Context, start, end uint64) (logs []*types.Log, blockNumber uint64, err error) {

	checkBlock := func(i uint64, header *types.Header) (logs []*types.Log, blockNumber uint64, err error) {
		// Get the logs of the block
		receipts, err := f.backend.GetReceipts(ctx, header.Hash())
		if err != nil {
			return nil, end, err
		}
		var unfiltered []*types.Log
		for _, receipt := range receipts {
			unfiltered = append(unfiltered, ([]*types.Log)(receipt.Logs)...)
		}
		logs = filterLogs(unfiltered, nil, nil, f.addresses, f.topics)
		if len(logs) > 0 {
			return logs, i, nil
		}
		return nil, i, nil
	}

	if f.useBloomBits {
		haveBloomBitsBefore := core.GetBloomBitsAvailable(f.db) * bloombits.SectionSize
		e := end
		if haveBloomBitsBefore <= e {
			e = haveBloomBitsBefore - 1
		}

		stop := make(chan struct{})
		defer close(stop)
		//fmt.Println("GetMatches")
		matches := f.matcher.GetMatches(start, e, stop)
		//fmt.Println("GetMatches ret")
		errChn := f.serveMatcher(ctx, stop)

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

				l, b, e := checkBlock(i, header)

				//fmt.Println("match", i, f.bloomFilter(header.Bloom), len(l))
				/*for i := 0; i < 16; i++ {
					fmt.Println(header.Bloom[i*16 : i*16+16])
				}*/

				if l != nil || e != nil {
					return l, b, e
				}
			case err := <-errChn:
				return logs, end, err
			case <-ctx.Done():
				return nil, end, ctx.Err()
			}
		}

		if end < haveBloomBitsBefore {
			return logs, end, nil
		} else {
			start = haveBloomBitsBefore
		}
	}

	for i := start; i <= end; i++ {
		blockNumber := rpc.BlockNumber(i)
		header, err := f.backend.HeaderByNumber(ctx, blockNumber)
		if header == nil || err != nil {
			return logs, end, err
		}

		// Use bloom filtering to see if this block is interesting given the
		// current parameters
		if f.bloomFilter(header.Bloom) {
			l, b, e := checkBlock(i, header)
			if l != nil || e != nil {
				return l, b, e
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
