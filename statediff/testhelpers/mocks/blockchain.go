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

	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// BlockChain is a mock blockchain for testing
type BlockChain struct {
	ParentHashesLookedUp []common.Hash
	parentBlocksToReturn []*types.Block
	callCount            int
	ChainEvents          []core.ChainEvent
}

// AddToStateDiffProcessedCollection mock method
func (blockChain *BlockChain) AddToStateDiffProcessedCollection(hash common.Hash) {}

// SetParentBlocksToReturn mock method
func (blockChain *BlockChain) SetParentBlocksToReturn(blocks []*types.Block) {
	blockChain.parentBlocksToReturn = blocks
}

// GetBlockByHash mock method
func (blockChain *BlockChain) GetBlockByHash(hash common.Hash) *types.Block {
	blockChain.ParentHashesLookedUp = append(blockChain.ParentHashesLookedUp, hash)

	var parentBlock *types.Block
	if len(blockChain.parentBlocksToReturn) > 0 {
		parentBlock = blockChain.parentBlocksToReturn[blockChain.callCount]
	}

	blockChain.callCount++
	return parentBlock
}

// SetChainEvents mock method
func (blockChain *BlockChain) SetChainEvents(chainEvents []core.ChainEvent) {
	blockChain.ChainEvents = chainEvents
}

// SubscribeChainEvent mock method
func (blockChain *BlockChain) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	subErr := errors.New("Subscription Error")

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
