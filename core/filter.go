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

package core

import (
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

type AccountChange struct {
	Address, StateAddress []byte
}

// Filtering interface
type Filter struct {
	eth      Backend
	earliest int64
	latest   int64
	skip     int
	address  []common.Address
	max      int
	topics   [][]common.Hash

	BlockCallback       func(*types.Block, state.Logs)
	TransactionCallback func(*types.Transaction)
	LogsCallback        func(state.Logs)
}

// Create a new filter which uses a bloom filter on blocks to figure out whether a particular block
// is interesting or not.
func NewFilter(eth Backend) *Filter {
	return &Filter{eth: eth}
}

// Set the earliest and latest block for filtering.
// -1 = latest block (i.e., the current block)
// hash = particular hash from-to
func (self *Filter) SetEarliestBlock(earliest int64) {
	self.earliest = earliest
}

func (self *Filter) SetLatestBlock(latest int64) {
	self.latest = latest
}

func (self *Filter) SetAddress(addr []common.Address) {
	self.address = addr
}

func (self *Filter) SetTopics(topics [][]common.Hash) {
	self.topics = topics
}

func (self *Filter) SetMax(max int) {
	self.max = max
}

func (self *Filter) SetSkip(skip int) {
	self.skip = skip
}

// Run filters logs with the current parameters set
func (self *Filter) Find() state.Logs {
	earliestBlock := self.eth.ChainManager().CurrentBlock()
	var earliestBlockNo uint64 = uint64(self.earliest)
	if self.earliest == -1 {
		earliestBlockNo = earliestBlock.NumberU64()
	}
	var latestBlockNo uint64 = uint64(self.latest)
	if self.latest == -1 {
		latestBlockNo = earliestBlock.NumberU64()
	}

	var (
		logs  state.Logs
		block = self.eth.ChainManager().GetBlockByNumber(latestBlockNo)
	)

done:
	for i := 0; block != nil; i++ {
		// Quit on latest
		switch {
		case block.NumberU64() == 0:
			break done
		case block.NumberU64() < earliestBlockNo:
			break done
		case self.max <= len(logs):
			break done
		}

		// Use bloom filtering to see if this block is interesting given the
		// current parameters
		if self.bloomFilter(block) {
			// Get the logs of the block
			unfiltered, err := self.eth.BlockProcessor().GetLogs(block)
			if err != nil {
				glog.V(logger.Warn).Infoln("err: filter get logs ", err)

				break
			}

			logs = append(logs, self.FilterLogs(unfiltered)...)
		}

		block = self.eth.ChainManager().GetBlock(block.ParentHash())
	}

	skip := int(math.Min(float64(len(logs)), float64(self.skip)))

	return logs[skip:]
}

func includes(addresses []common.Address, a common.Address) bool {
	for _, addr := range addresses {
		if addr != a {
			return false
		}
	}

	return true
}

func (self *Filter) FilterLogs(logs state.Logs) state.Logs {
	var ret state.Logs

	// Filter the logs for interesting stuff
Logs:
	for _, log := range logs {
		if len(self.address) > 0 && !includes(self.address, log.Address) {
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
	if len(self.address) > 0 {
		var included bool
		for _, addr := range self.address {
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
