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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
)

type AccountChange struct {
	Address, StateAddress []byte
}

// Filtering interface
type Filter struct {
	created time.Time

	db         ethdb.Database
	begin, end int64
	addresses  []common.Address
	topics     [][]common.Hash

	BlockCallback       func(*types.Block, vm.Logs)
	TransactionCallback func(*types.Transaction)
	LogCallback         func(*vm.Log, bool)
}

// Create a new filter which uses a bloom filter on blocks to figure out whether a particular block
// is interesting or not.
func New(db ethdb.Database) *Filter {
	return &Filter{db: db}
}

// Set the earliest and latest block for filtering.
// -1 = latest block (i.e., the current block)
// hash = particular hash from-to
func (self *Filter) SetBeginBlock(begin int64) {
	self.begin = begin
}

func (self *Filter) SetEndBlock(end int64) {
	self.end = end
}

func (self *Filter) SetAddresses(addr []common.Address) {
	self.addresses = addr
}

func (self *Filter) SetTopics(topics [][]common.Hash) {
	self.topics = topics
}

// Run filters logs with the current parameters set
func (self *Filter) Find() vm.Logs {
	latestHash := core.GetHeadBlockHash(self.db)
	latestBlock := core.GetBlock(self.db, latestHash, core.GetBlockNumber(self.db, latestHash))
	var beginBlockNo uint64 = uint64(self.begin)
	if self.begin == -1 {
		beginBlockNo = latestBlock.NumberU64()
	}
	var endBlockNo uint64 = uint64(self.end)
	if self.end == -1 {
		endBlockNo = latestBlock.NumberU64()
	}

	// if no addresses are present we can't make use of fast search which
	// uses the mipmap bloom filters to check for fast inclusion and uses
	// higher range probability in order to ensure at least a false positive
	if len(self.addresses) == 0 {
		return self.getLogs(beginBlockNo, endBlockNo)
	}
	return self.mipFind(beginBlockNo, endBlockNo, 0)
}

func (self *Filter) mipFind(start, end uint64, depth int) (logs vm.Logs) {
	level := core.MIPMapLevels[depth]
	// normalise numerator so we can work in level specific batches and
	// work with the proper range checks
	for num := start / level * level; num <= end; num += level {
		// find addresses in bloom filters
		bloom := core.GetMipmapBloom(self.db, num, level)
		for _, addr := range self.addresses {
			if bloom.TestBytes(addr[:]) {
				// range check normalised values and make sure that
				// we're resolving the correct range instead of the
				// normalised values.
				start := uint64(math.Max(float64(num), float64(start)))
				end := uint64(math.Min(float64(num+level-1), float64(end)))
				if depth+1 == len(core.MIPMapLevels) {
					logs = append(logs, self.getLogs(start, end)...)
				} else {
					logs = append(logs, self.mipFind(start, end, depth+1)...)
				}
				// break so we don't check the same range for each
				// possible address. Checks on multiple addresses
				// are handled further down the stack.
				break
			}
		}
	}

	return logs
}

func (self *Filter) getLogs(start, end uint64) (logs vm.Logs) {
	var block *types.Block

	for i := start; i <= end; i++ {
		hash := core.GetCanonicalHash(self.db, i)
		if hash != (common.Hash{}) {
			block = core.GetBlock(self.db, hash, i)
		} else { // block not found
			return logs
		}

		// Use bloom filtering to see if this block is interesting given the
		// current parameters
		if self.bloomFilter(block) {
			// Get the logs of the block
			var (
				receipts   = core.GetBlockReceipts(self.db, block.Hash(), i)
				unfiltered vm.Logs
			)
			for _, receipt := range receipts {
				unfiltered = append(unfiltered, receipt.Logs...)
			}
			logs = append(logs, self.FilterLogs(unfiltered)...)
		}
	}

	return logs
}

func includes(addresses []common.Address, a common.Address) bool {
	for _, addr := range addresses {
		if addr == a {
			return true
		}
	}

	return false
}

func (self *Filter) FilterLogs(logs vm.Logs) vm.Logs {
	var ret vm.Logs

	// Filter the logs for interesting stuff
Logs:
	for _, log := range logs {
		if len(self.addresses) > 0 && !includes(self.addresses, log.Address) {
			continue
		}

		logTopics := make([]common.Hash, len(self.topics))
		copy(logTopics, log.Topics)

		// If the to filtered topics is greater than the amount of topics in
		//  logs, skip.
		if len(self.topics) > len(log.Topics) {
			continue Logs
		}

		for i, topics := range self.topics {
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

func (self *Filter) bloomFilter(block *types.Block) bool {
	if len(self.addresses) > 0 {
		var included bool
		for _, addr := range self.addresses {
			if types.BloomLookup(block.Bloom(), addr) {
				included = true
				break
			}
		}

		if !included {
			return false
		}
	}

	for _, sub := range self.topics {
		var included bool
		for _, topic := range sub {
			if (topic == common.Hash{}) || types.BloomLookup(block.Bloom(), topic) {
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
