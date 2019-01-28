package mocks

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

type BlockChain struct {
	ParentHashesLookedUp []common.Hash
	parentBlocksToReturn []*types.Block
	callCount            int
}

func (mc *BlockChain) SetParentBlockToReturn(blocks []*types.Block) {
	mc.parentBlocksToReturn = blocks
}

func (mc *BlockChain) GetBlockByHash(hash common.Hash) *types.Block {
	mc.ParentHashesLookedUp = append(mc.ParentHashesLookedUp, hash)

	var parentBlock types.Block
	if len(mc.parentBlocksToReturn) > 0 {
		parentBlock = *mc.parentBlocksToReturn[mc.callCount]
	}

	mc.callCount++
	return &parentBlock
}

func (BlockChain) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	panic("implement me")
}
