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
	"github.com/ethereum/go-ethereum/log"
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
	trackedMetricName = "eth/block/reward/"
)

var (
	standardBlockGauge = metrics.NewRegisteredGauge("eth/block/standard", nil)
	mevBlockGauge      = metrics.NewRegisteredGauge("eth/block/mev", nil)
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
	name = fmt.Sprintf("%s%s", trackedMetricName, name)

	sampler := func() metrics.Sample {
		return metrics.ResettingSample(
			metrics.NewExpDecaySample(1028, 0.015),
		)
	}
	gwei := new(big.Int).Quo(reward, big.NewInt(params.GWei))
	metrics.GetOrRegisterHistogramLazy(name, nil, sampler).Update(gwei.Int64())

	log.Info("Tracked block reward", "duration(s)", genTime, "base", base, "reward(gwei)", gwei)
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
	signer  types.Signer
	closed  chan struct{}
	wg      sync.WaitGroup
}

// RegisterRewardBench registers the reward benchmark service.
func RegisterRewardBench(stack *node.Node, backend *eth.Ethereum) *RewardBench {
	bench := &RewardBench{
		api:     catalyst.NewConsensusAPI(backend),
		chain:   backend.BlockChain(),
		rewards: make(map[uint64][]*big.Int),
		signer:  types.LatestSigner(backend.BlockChain().Config()),
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

// isMEVBlock returns an indicator that the provided block is built along with
// mev-boost rules. Usually mev-block will append a payment transaction to
// validator at the end of block. It's very hacky to determine block in such
// approach, todo(rjl493456442) any other better approach?
func (bench *RewardBench) isMEVBlock(block *types.Block) (bool, error) {
	txs := block.Transactions()
	if len(txs) == 0 {
		return false, nil
	}
	sender, err := bench.signer.Sender(txs[len(txs)-1])
	if err != nil {
		return false, err
	}
	return sender == block.Coinbase(), nil
}

// readReward retrieves the real reward earned by the fee recipient in the specified
// block. The reward includes the standard transaction fees plus some additional mev
// revenue.
func (bench *RewardBench) readReward(block *types.Block) (*big.Int, error) {
	isMEV, err := bench.isMEVBlock(block)
	if err != nil {
		return nil, err
	}
	if isMEV {
		mevBlockGauge.Inc(1)
		payment := block.Transactions()[len(block.Transactions())-1]
		return payment.Value(), nil
	}
	standardBlockGauge.Inc(1)

	var (
		fees     = new(big.Int)
		txs      = block.Transactions()
		receipts = bench.chain.GetReceiptsByHash(block.Hash())
	)
	if len(receipts) != len(txs) {
		return nil, errors.New("invalid block")
	}
	for i, tx := range txs {
		price := tx.EffectiveGasTipValue(block.BaseFee())
		fees = new(big.Int).Add(fees, new(big.Int).Mul(price, big.NewInt(int64(receipts[i].GasUsed))))
	}
	return fees, nil
}

func (bench *RewardBench) process(head *types.Block, done chan struct{}) {
	defer close(done)

	for number, rewards := range bench.rewards {
		if number > head.NumberU64() {
			continue
		}
		block := bench.chain.GetBlockByNumber(number)
		if block == nil {
			delete(bench.rewards, number)
			continue
		}
		base, err := bench.readReward(block)
		if err != nil {
			delete(bench.rewards, number)
			continue
		}
		trackRewards(append(rewards, base))
	}
	// Calculate the block rewards of next block locally.
	// This function will be blocked for a few seconds
	// in order to collect as much statistics as possible.
	local, err := bench.calcRewards(head)
	if err != nil {
		return
	}
	bench.rewards[head.NumberU64()+1] = local
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
