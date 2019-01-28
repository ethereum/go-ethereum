// Copyright 2019 The go-ethereum Authors
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

package mocks

import (
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/state"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// BlockChain is a mock blockchain for testing
type BlockChain struct {
	HashesLookedUp         []common.Hash
	blocksToReturnByHash   map[common.Hash]*types.Block
	blocksToReturnByNumber map[uint64]*types.Block
	callCount              int
	ChainEvents            []core.ChainEvent
	Receipts               map[common.Hash]types.Receipts
	TDByHash               map[common.Hash]*big.Int
}

// SetBlocksForHashes mock method
func (blockChain *BlockChain) SetBlocksForHashes(blocks map[common.Hash]*types.Block) {
	if blockChain.blocksToReturnByHash == nil {
		blockChain.blocksToReturnByHash = make(map[common.Hash]*types.Block)
	}
	blockChain.blocksToReturnByHash = blocks
}

// GetBlockByHash mock method
func (blockChain *BlockChain) GetBlockByHash(hash common.Hash) *types.Block {
	blockChain.HashesLookedUp = append(blockChain.HashesLookedUp, hash)

	var block *types.Block
	if len(blockChain.blocksToReturnByHash) > 0 {
		block = blockChain.blocksToReturnByHash[hash]
	}

	return block
}

// SetChainEvents mock method
func (blockChain *BlockChain) SetChainEvents(chainEvents []core.ChainEvent) {
	blockChain.ChainEvents = chainEvents
}

// SubscribeChainEvent mock method
func (blockChain *BlockChain) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	subErr := errors.New("subscription error")

	var eventCounter int
	subscription := event.NewSubscription(func(quit <-chan struct{}) error {
		for _, chainEvent := range blockChain.ChainEvents {
			if eventCounter > 1 {
				time.Sleep(250 * time.Millisecond)
				return subErr
			}
			select {
			case ch <- chainEvent:
			case <-quit:
				return nil
			}
			eventCounter++
		}
		return nil
	})

	return subscription
}

// SetReceiptsForHash test method
func (blockChain *BlockChain) SetReceiptsForHash(hash common.Hash, receipts types.Receipts) {
	if blockChain.Receipts == nil {
		blockChain.Receipts = make(map[common.Hash]types.Receipts)
	}
	blockChain.Receipts[hash] = receipts
}

// GetReceiptsByHash mock method
func (blockChain *BlockChain) GetReceiptsByHash(hash common.Hash) types.Receipts {
	return blockChain.Receipts[hash]
}

// SetBlockForNumber test method
func (blockChain *BlockChain) SetBlockForNumber(block *types.Block, number uint64) {
	if blockChain.blocksToReturnByNumber == nil {
		blockChain.blocksToReturnByNumber = make(map[uint64]*types.Block)
	}
	blockChain.blocksToReturnByNumber[number] = block
}

// GetBlockByNumber mock method
func (blockChain *BlockChain) GetBlockByNumber(number uint64) *types.Block {
	return blockChain.blocksToReturnByNumber[number]
}

// GetTdByHash mock method
func (blockChain *BlockChain) GetTdByHash(hash common.Hash) *big.Int {
	return blockChain.TDByHash[hash]
}

func (blockChain *BlockChain) SetTdByHash(hash common.Hash, td *big.Int) {
	if blockChain.TDByHash == nil {
		blockChain.TDByHash = make(map[common.Hash]*big.Int)
	}
	blockChain.TDByHash[hash] = td
}

func (blockChain *BlockChain) UnlockTrie(root common.Hash) {}

func (BlockChain *BlockChain) StateCache() state.Database {
	return nil
}
