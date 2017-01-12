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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/tylertreat/BoomFilters"
	"golang.org/x/net/context"
)

// Backend defines the interface the filter packages needs to retrieve logs.
type Backend interface {
	ChainDb() ethdb.Database
	EventMux() *event.TypeMux
	HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error)
	GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error)
}

// Filter can be used to search for particular logs
type Filter struct {
	be        Backend
	crit      FilterCriteria
	useMipMap bool
}

// filterRange helper structure that keeps track of a bloom filter and the range it covers.
type filterRange struct {
	begin uint64
	end   uint64
	bloom boom.Filter
}

// New creates a filter that can be used to search for logs that match the given criteria.
// When `useMipMap` is `true` it uses pre-calculated bloom filters to optimize searching.
// These pre-calculated bloom filters are not available in light mode and require at least
// one address in the criteria.
func New(be Backend, crit FilterCriteria, useMipMap bool) *Filter {
	if crit.FromBlock == nil || crit.FromBlock.Int64() < 0 {
		crit.FromBlock = big.NewInt(0)
	}
	return &Filter{
		be:        be,
		useMipMap: useMipMap,
		crit:      crit,
	}
}

// Find returns all logs matching the filter criteria.
func (f *Filter) Find(ctx context.Context) ([]*types.Log, error) {
	if f.useMipMap && len(f.crit.Addresses) > 0 {
		return f.findByAddressIdx(ctx)
	}
	return f.findFullRangeScan(ctx)
}

// findByAddressIdx returns all logs matching the criteria. It uses an bloom filter based
// index on log addresses for faster searching (not available in light mode).
func (f *Filter) findByAddressIdx(ctx context.Context) ([]*types.Log, error) {
	var (
		depth      = 0
		start, end uint64
		level      = core.AddrBloomMapLevels[depth]
		logs       []*types.Log
		filters    = []filterRange{}
	)

	start = f.crit.FromBlock.Uint64()

	if f.crit.ToBlock.Int64() < 0 {
		h, err := f.be.HeaderByNumber(ctx, rpc.BlockNumber(f.crit.ToBlock.Int64()))
		if err != nil {
			return nil, err
		}
		end = h.Number.Uint64()
	} else {
		end = f.crit.ToBlock.Uint64()
	}

	// find top level block ranges that (probably) contains logs from one of the target addresses.
	for num := (start / level) * level; num <= end; num += level {
		_, bloom, err := core.GetBloomLogs(f.be.ChainDb(), num, depth)
		if err != nil {
			return logs, nil
		}
		if f.matchAddress(bloom) {
			filters = append(filters, filterRange{num, num + core.AddrBloomMapLevels[depth], bloom})
		}
	}

	return f.filterRange(ctx, depth+1, filters, start, end)
}

// filterRange returns the set of logs that match the filter criteria in bf within the block range
// [start, end]. This is a depth first search to guarantee logs are sorted by block number.
func (f *Filter) filterRange(ctx context.Context, depth int, filters []filterRange, start, end uint64) ([]*types.Log, error) {
	var logs []*types.Log
	for _, filter := range filters {
		// reached lowest level, do a full block scan on the filter range
		if depth == len(core.AddrBloomMapLevels) {
			return f.scanRangeWithFilter(ctx, filter, start, end)
		}

		// narrow search by evicting branches that are guaranteed not to include logs.
		level := core.AddrBloomMapLevels[depth]
		for num := (filter.begin / level) * level; num <= end && num < filter.end; num += level {
			if num+level < start {
				continue
			}
			_, bloom, err := core.GetBloomLogs(f.be.ChainDb(), num, depth)
			if err != nil {
				return nil, err
			}

			if f.matchAddress(bloom) {
				fr := filterRange{num, num + level, bloom}
				l, err := f.filterRange(ctx, depth+1, []filterRange{fr}, start, end)
				if err != nil {
					return nil, err
				}
				logs = append(logs, l...)
			}
		}
	}
	return logs, nil
}

// logs is a helper that returns all logs in the specified block that matches the filter criteria.
func (f *Filter) logs(ctx context.Context, blockNum uint64) ([]*types.Log, error) {
	var logs []*types.Log
	blockNumber := rpc.BlockNumber(blockNum)
	header, err := f.be.HeaderByNumber(ctx, blockNumber)
	if header == nil || err != nil {
		return logs, err
	}

	if BloomFilter(header.Bloom, f.crit.Addresses, f.crit.Topics) {
		receipts, err := f.be.GetReceipts(ctx, header.Hash())
		if err != nil {
			return nil, err
		}

		var unfiltered []*types.Log
		for _, receipt := range receipts {
			unfiltered = append(unfiltered, ([]*types.Log)(receipt.Logs)...)
		}
		logs = append(logs, filterLogs(unfiltered, nil, nil, f.crit.Addresses, f.crit.Topics)...)
	}
	return logs, nil
}

// scanRangeWithFilter returns all logs matching the bf filter criteria.
func (f *Filter) scanRangeWithFilter(ctx context.Context, filter filterRange, start, end uint64) ([]*types.Log, error) {
	if start < filter.begin {
		start = filter.begin
	}
	// in case the toBlock is within the filter range make it inclusive.
	if end > filter.begin && end <= filter.end {
		end += 1
	} else if end > filter.end {
		end = filter.end
	}

	var logs []*types.Log
	for i := start; i < end; i++ {
		if l, err := f.logs(ctx, i); err == nil {
			logs = append(logs, l...)
		} else {
			return nil, err
		}
	}

	return logs, nil
}

// findFullRangeScan performance a naive full blocks scan in the range [start, end] and
// returns logs matching the filters criteria.
func (f *Filter) findFullRangeScan(ctx context.Context) ([]*types.Log, error) {
	var (
		start = f.crit.FromBlock.Uint64()
		end   uint64
		logs  []*types.Log
	)

	if f.crit.ToBlock.Int64() < 0 {
		h, err := f.be.HeaderByNumber(ctx, rpc.BlockNumber(f.crit.ToBlock.Int64()))
		if err != nil {
			return nil, err
		}
		end = h.Number.Uint64()
	} else {
		end = f.crit.ToBlock.Uint64()
	}

	for i := start; i < end; i++ {
		if l, err := f.logs(ctx, i); err == nil {
			logs = append(logs, l...)
		} else {
			return nil, err
		}
	}
	return logs, nil
}

// matchAddress returns an indication if at least one of the addresses
// in bf.addresses is stored in the given bloom.
func (f *Filter) matchAddress(bloom boom.Filter) bool {
	for _, addr := range f.crit.Addresses {
		if bloom.Test(addr.Bytes()) {
			return true
		}
	}
	return false
}

// includes returns an indication if a is included in addresses.
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

// BloomFilter returns an indication if the given addresses and topics match with the given bloom
func BloomFilter(bloom types.Bloom, addresses []common.Address, topics [][]common.Hash) bool {
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
