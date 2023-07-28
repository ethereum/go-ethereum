// Copyright 2023 The go-ethereum Authors
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

package ethapi

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/supply"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

// GethAPI is the collection of geth-specific APIs exposed over the geth
// namespace.
type GethAPI struct {
	b Backend
}

// NewDebugAPI creates a new instance of DebugAPI.
func NewGethAPI(b Backend) *GethAPI {
	return &GethAPI{b: b}
}

// SupplyDelta send a notification each time a new block is appended to the chain
// with various counters about Ether supply delta: the state diff (if
// available), block and uncle subsidy, 1559 burn.
func (api *GethAPI) SupplyDelta(ctx context.Context, from uint64) (*rpc.Subscription, error) {
	// If supply delta tracking is not explcitly enabled, refuse to service this
	// endpoint. Although we could enable the simple calculations, it might
	// end up as an unexpected load on RPC providers, so let's not surprise.
	if !api.b.Config().EnableSupplyDeltaRecording {
		return nil, errors.New("supply delta recording not enabled")
	}
	config := api.b.ChainConfig()

	// Supply delta recording enabled, create a subscription to stream through
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}
	rpcSub := notifier.CreateSubscription()

	// Define an internal type for supply delta notifications
	type supplyDeltaNotification struct {
		Number      uint64      `json:"block"`
		Hash        common.Hash `json:"hash"`
		ParentHash  common.Hash `json:"parentHash"`
		SupplyDelta *big.Int    `json:"supplyDelta"`
		Reward      *big.Int    `json:"reward"`
		Withdrawals *big.Int    `json:"withdrawals"`
		Burn        *big.Int    `json:"burn"`
		Destruct    *big.Int    `json:"destruct"`
	}
	// Define a method to convert a block into an supply delta notification
	service := func(block *types.Block) {
		// Retrieve the state-crawled supply delta - if available
		crawled := rawdb.ReadSupplyDelta(api.b.ChainDb(), block.NumberU64(), block.Hash())

		// Calculate the issuance and burn from the block's contents
		rewards, withdrawals := supply.Issuance(block, config)
		burn := supply.Burn(block.Header())

		// Calculate the difference between the "calculated" and "crawled" supply delta
		var diff *big.Int
		if crawled != nil {
			diff = new(big.Int).Set(crawled)
			diff.Sub(diff, rewards)
			diff.Sub(diff, withdrawals)
			diff.Add(diff, burn)
		}
		// Push the supply delta to the user
		notifier.Notify(rpcSub.ID, &supplyDeltaNotification{
			Number:      block.NumberU64(),
			Hash:        block.Hash(),
			ParentHash:  block.ParentHash(),
			SupplyDelta: crawled,
			Reward:      rewards,
			Withdrawals: withdrawals,
			Burn:        burn,
			Destruct:    diff,
		})
	}
	go func() {
		// Iterate over all blocks from the requested source up to head and push
		// out historical supply delta values to the user. Checking the head after
		// each iteration is a bit heavy, but it's not really relevant compared
		// to pulling blocks from disk, so this keeps thing simpler to switch
		// from historical blocks to live blocks.
		for number := from; number <= api.b.CurrentBlock().Number.Uint64(); number++ {
			block := rawdb.ReadBlock(api.b.ChainDb(), rawdb.ReadCanonicalHash(api.b.ChainDb(), number), number)
			if block == nil {
				log.Error("Missing block for supply delta reporting", "number", number)
				return
			}
			service(block)
		}
		// Subscribe to chain events and keep emitting supply deltas on all
		// branches
		canonBlocks := make(chan core.ChainEvent)
		canonBlocksSub := api.b.SubscribeChainEvent(canonBlocks)
		defer canonBlocksSub.Unsubscribe()

		sideBlocks := make(chan core.ChainSideEvent)
		sideBlocksSub := api.b.SubscribeChainSideEvent(sideBlocks)
		defer sideBlocksSub.Unsubscribe()

		for {
			select {
			case event := <-canonBlocks:
				service(event.Block)
			case event := <-sideBlocks:
				service(event.Block)
			case <-rpcSub.Err():
				return
			case <-notifier.Closed():
				return
			}
		}
	}()
	return rpcSub, nil
}
