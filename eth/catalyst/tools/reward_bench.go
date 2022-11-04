// Copyright 2022 The go-ethereum Authors
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

package tools

import (
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/beacon"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
)

const (
	// payloadWaitTime defines the maximum time allowed to wait for
	// payload generation. Usually 6-8 seconds is given by consensus
	// layer to generate a payload.
	payloadWaitTime = time.Second * 8

	// recommitInterval defines the default recommit time interval.
	recommitInterval = 2

	// trackedMetricName is the prefix of the block reward tracking.
	trackedMetricName = "core/reward/"
)

func track(reward *big.Int, base bool, genTime int) {
	if !metrics.Enabled {
		return
	}
	var name string
	if base {
		name = "base"
	} else {
		name = fmt.Sprintf("%ds", genTime)
	}
	name = fmt.Sprintf("%s/%s", trackedMetricName, name)
	metrics.GetOrRegisterHistogram(name, nil, metrics.NewExpDecaySample(1028, 0.015)).Update(new(big.Int).Quo(reward, big.NewInt(params.GWei)).Int64())
}

func trackRewards(rewards []*big.Int) {
	if len(rewards) < 1 {
		return // reject invalid
	}
	for i, reward := range rewards {
		track(reward, i == len(rewards)-1, i*recommitInterval)
	}
}

// RewardBench is the development service to compare the block revenue.
// It uses the real revenue of network blocks as the baseline, compares
// all local computed revenues with different time allowance.
type RewardBench struct {
	api     *catalyst.ConsensusAPI
	chain   *core.BlockChain
	rewards map[uint64][]*big.Int
	closed  chan struct{}
	wg      sync.WaitGroup
}

// RegisterRewardBench registers the reward benchmark service.
func RegisterRewardBench(stack *node.Node, backend *eth.Ethereum) *RewardBench {
	bench := &RewardBench{
		api:     catalyst.NewConsensusAPI(backend),
		chain:   backend.BlockChain(),
		rewards: make(map[uint64][]*big.Int),
		closed:  make(chan struct{}),
	}
	stack.RegisterLifecycle(bench)
	return bench
}

// calcReward calculates a list of block rewards of the next block.
func (bench *RewardBench) calcRewards(block *types.Block) ([]*big.Int, error) {
	var (
		fcu = beacon.ForkchoiceStateV1{
			HeadBlockHash: block.Hash(),

			// These fields below are deliberately left as empty,
			// don't intent to modify the chain status at all.
			FinalizedBlockHash: common.Hash{},
			SafeBlockHash:      common.Hash{},
		}
		attrs = &beacon.PayloadAttributesV1{
			Timestamp:             uint64(time.Now().Unix()),
			Random:                common.Hash{},
			SuggestedFeeRecipient: common.HexToAddress("0xdeadbeef"),
		}
	)
	response, err := bench.api.ForkchoiceUpdatedV1(fcu, attrs)
	if err != nil {
		return nil, err
	}
	if response.PayloadID == nil {
		return nil, errors.New("failed to generate payload")
	}
	time.Sleep(payloadWaitTime)
	_, fees, feeHistory, err := bench.api.GetPayloadWithFees(*response.PayloadID)
	if err != nil {
		return nil, err
	}
	return append(feeHistory, fees), nil
}

// readReward retrieves the real reward earned by the fee recipient in the specified
// block. The reward includes the standard transaction fees plus some additional mev
// revenue.
func (bench *RewardBench) readReward(block *types.Block) (*big.Int, error) {
	parent := bench.chain.GetBlock(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return nil, errors.New("parent block is not found")
	}
	prestate, err := bench.chain.StateAt(parent.Root())
	if err != nil {
		return nil, err
	}
	poststate, err := bench.chain.StateAt(block.Root())
	if err != nil {
		return nil, err
	}
	return big.NewInt(0).Sub(poststate.GetBalance(block.Coinbase()), prestate.GetBalance(block.Coinbase())), nil
}

func (bench *RewardBench) process(block *types.Block, done chan struct{}) {
	defer close(done)

	rewards, ok := bench.rewards[block.NumberU64()]
	if ok {
		base, err := bench.readReward(block)
		if err != nil {
			return
		}
		trackRewards(append(rewards, base))
	}
	// Clean up the outdated block rewards.
	for number := range bench.rewards {
		if number <= block.NumberU64() {
			delete(bench.rewards, number)
		}
	}
	// Calculate the block rewards of next block locally.
	// This function will be blocked for a few seconds
	// in order to collect as many statistics as possible.
	local, err := bench.calcRewards(block)
	if err != nil {
		return
	}
	bench.rewards[block.NumberU64()+1] = local
}

// Start launches the reward benchmarker.
func (bench *RewardBench) Start() error {
	bench.wg.Add(1)
	go func() {
		defer bench.wg.Done()

		var (
			done   chan struct{}                       // Non-nil if background thread is active.
			events = make(chan core.ChainHeadEvent, 1) // Buffered to avoid locking up the event feed
		)
		sub := bench.chain.SubscribeChainHeadEvent(events)
		defer sub.Unsubscribe()

		for {
			select {
			case ev := <-events:
				if done == nil {
					done = make(chan struct{})
					go bench.process(ev.Block, done)
				}
			case <-done:
				done = nil
			case <-bench.closed:
				return
			}
		}
	}()
	return nil
}

// Stop stops the reward benchmarker to stop all background activities.
// This function can only be called for one time.
func (bench *RewardBench) Stop() error {
	close(bench.closed)
	bench.wg.Wait()
	return nil
}
