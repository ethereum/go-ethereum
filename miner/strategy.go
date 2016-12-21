// Copyright 2016 The go-ethereum Authors
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

package miner

import (
	"errors"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Strategy is a collection of optional callback methods that miner strategies
// may implement to influence the behavior of the miner.
type Strategy struct {
	Name string // Name of the strategy for logging purposes

	// OnMinedBlock is invoked when a new block is mined by the miner.
	OnMinedBlock func(emux *event.TypeMux, chain *core.BlockChain, pool *core.TxPool, block *types.Block) error
}

// NewZeroDiffStrategy creates a mining strategy that aims to push the difficulty
// of the blocks down to zero. It tries to achive this by delaying found blocks
// to always be above the configured block time limit, pushing the difficulty
// down a bit after every block.
//
// The strategy has a few nice properties that is orthogonal both to multiple
// zero-diff miners as well as other plain miners:
//
//  * The zero-diff miner may be ran with arbitrarilly many threads: if a block
//    is found faster than the target block time, it will be delayed anyway, so
//    the mining threads will go idle. If on the other hand the difficulty is
//    pushed up by a rouge miner and abandoned, the multiple threads will allow
//    pulling the difficulty down faster until it reaches sub-target times.
//  * The zero-diff miner can play along nicely with non zero-diff miners, since
//    it will either delay it's block to above-target times, or outright discard
//    its own block if another is found, thereby ensuring it only ever reduces
//    the difficulty, never increases.
//  * The zero-diff miner can also play along nicely with other zero-diff miners
//    by simulating block times at random between [target, 1.1*target], allowing
//    multiple zero-diff miners to co-exist and share blocks, without racing each
//    other for blocks and leading to a high uncle rate.
//
// Note, this strategy is only meaningful in trusted private networks where the
// goal of mining is not to secure the network, rather to provide a stable but
// resource-light testbed.
func NewZeroDiffStrategy(instantTxs bool, minBalance *big.Int) *Strategy {
	return &Strategy{
		Name: "zero-diff",

		OnMinedBlock: func(emux *event.TypeMux, chain *core.BlockChain, pool *core.TxPool, block *types.Block) error {
			// A new block was mined, calculate the required delay
			elapsed := time.Since(time.Unix(block.Time().Int64(), 0))
			delay := float64(15*time.Second)*(1.0+rand.Float64()/10.0) - float64(elapsed)

			// If the delay is negative, block times are way over the target already, release
			if delay <= 0 {
				return nil
			}
			// If instant transactions are enabled and the block contains something, release
			if instantTxs && len(block.Transactions()) > 0 {
				return nil
			}
			// If minimum balance requirements are set but not met, release
			if minBalance != nil {
				if state, _ := chain.State(); state.GetBalance(block.Coinbase()).Cmp(minBalance) < 0 {
					return nil
				}
			}
			// Otherwise monitor head and transaction events
			head := emux.Subscribe(core.ChainHeadEvent{})
			defer head.Unsubscribe()

			txs := emux.Subscribe(core.TxPreEvent{})
			defer txs.Unsubscribe()

			// Double check for events that happened before subscriptions
			if chain.CurrentHeader().Hash() != block.ParentHash() {
				return errors.New("stale block") // Another block arrived already, drop this
			}
			if pend, _ := pool.Stats(); pend > 0 {
				return nil // Transaction pending in the pool, release this and start processing them
			}
			// Wait for some event to occur that releases or bins the block
			select {
			case <-time.After(time.Duration(delay)):
				return nil // Timeout for difficulty reduction reached, release
			case <-head.Chan():
				return errors.New("concurrent block") // Alternate block arrived, drop this
			case <-txs.Chan():
				return nil // Transaction arrived, include it as fast as possible
			}
		},
	}
}
