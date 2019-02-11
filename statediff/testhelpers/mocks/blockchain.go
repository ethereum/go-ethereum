package mocks

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

type BlockChain struct {
	ParentHashesLookedUp []common.Hash
	parentBlocksToReturn []*types.Block
	callCount            int
	ChainEvents          []core.ChainEvent
}

func (mc *BlockChain) SetParentBlocksToReturn(blocks []*types.Block) {
	mc.parentBlocksToReturn = blocks
}

func (mc *BlockChain) GetBlockByHash(hash common.Hash) *types.Block {
	mc.ParentHashesLookedUp = append(mc.ParentHashesLookedUp, hash)

	var parentBlock *types.Block
	if len(mc.parentBlocksToReturn) > 0 {
		parentBlock = mc.parentBlocksToReturn[mc.callCount]
	}

	mc.callCount++
	return parentBlock
}

func (bc *BlockChain) SetChainEvents(chainEvents []core.ChainEvent) {
	bc.ChainEvents = chainEvents
}

func (bc *BlockChain) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	subErr := errors.New("Subscription Error")

	var eventCounter int
	subscription := event.NewSubscription(func(quit <-chan struct{}) error {
		for _, chainEvent := range bc.ChainEvents {
			if eventCounter > 1 {
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
