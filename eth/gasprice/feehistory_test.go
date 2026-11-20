// Copyright 2021 The go-ethereum Authors
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

package gasprice

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

// Delay one header in virtual time so the other result is consumed first.
type feeHistoryOrderBackend struct {
	OracleBackend
	config  *params.ChainConfig
	headers map[rpc.BlockNumber]*types.Header
	delayed rpc.BlockNumber
	last    rpc.BlockNumber
}

func (b *feeHistoryOrderBackend) ChainConfig() *params.ChainConfig { return b.config }

func (b *feeHistoryOrderBackend) HeaderByNumber(_ context.Context, number rpc.BlockNumber) (*types.Header, error) {
	if number == rpc.LatestBlockNumber {
		return &types.Header{Number: big.NewInt(int64(b.last))}, nil
	}
	if number == b.delayed {
		time.Sleep(time.Second)
	}
	return b.headers[number], nil
}

func TestFeeHistoryBlobForkBoundary(t *testing.T) {
	for _, delayed := range []rpc.BlockNumber{99, 100} {
		for _, missingLast := range []bool{false, true} {
			t.Run(fmt.Sprintf("delayed=%d/missingLast=%t", delayed, missingLast), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					zero, cancun := uint64(0), uint64(1000)
					backend := &feeHistoryOrderBackend{
						config: &params.ChainConfig{
							LondonBlock: big.NewInt(0), ShanghaiTime: &zero, CancunTime: &cancun,
							BlobScheduleConfig: params.DefaultBlobSchedule,
						},
						headers: map[rpc.BlockNumber]*types.Header{
							99:  {Number: big.NewInt(99), Time: 990, BaseFee: big.NewInt(8), GasLimit: 30000000, GasUsed: 15000000},
							100: {Number: big.NewInt(100), Time: 1000, BaseFee: big.NewInt(8), GasLimit: 30000000, GasUsed: 15000000, ExcessBlobGas: &zero, BlobGasUsed: &zero},
						},
						delayed: delayed,
						last:    100,
					}
					if missingLast {
						// Simulate a head that disappears while retrieving the range.
						backend.last = 101
					}
					oracle := &Oracle{
						backend: backend, maxHeaderHistory: 3,
						historyCache: lru.NewCache[cacheKey, processedFees](3),
					}
					first, _, baseFee, ratio, blobFee, blobRatio, err := oracle.FeeHistory(context.Background(), uint64(backend.last-98), backend.last, nil)
					if err != nil {
						t.Fatal(err)
					}
					if first.Uint64() != 99 || len(baseFee) != 3 || len(blobFee) != 3 || len(ratio) != 2 || len(blobRatio) != 2 {
						t.Fatalf("unexpected range: first=%v baseFee=%v blobFee=%v ratio=%v blobRatio=%v", first, baseFee, blobFee, ratio, blobRatio)
					}
					for i, want := range []int64{0, 1, 1} {
						if blobFee[i] == nil || blobFee[i].Cmp(big.NewInt(want)) != 0 {
							t.Errorf("blob fee %d: have %v, want %d", i, blobFee[i], want)
						}
						if baseFee[i] == nil || baseFee[i].Cmp(big.NewInt(8)) != 0 {
							t.Errorf("base fee %d: have %v, want 8", i, baseFee[i])
						}
					}
				})
			})
		}
	}
}

func TestFeeHistory(t *testing.T) {
	var cases = []struct {
		pending             bool
		maxHeader, maxBlock uint64
		count               uint64
		last                rpc.BlockNumber
		percent             []float64
		expFirst            uint64
		expCount            int
		expErr              error
	}{
		{false, 1000, 1000, 10, 30, nil, 21, 10, nil},
		{false, 1000, 1000, 10, 30, []float64{0, 10}, 21, 10, nil},
		{false, 1000, 1000, 10, 30, []float64{20, 10}, 0, 0, errInvalidPercentile},
		{false, 1000, 1000, 1000000000, 30, nil, 0, 31, nil},
		{false, 1000, 1000, 1000000000, rpc.LatestBlockNumber, nil, 0, 33, nil},
		{false, 1000, 1000, 10, 40, nil, 0, 0, errRequestBeyondHead},
		{true, 1000, 1000, 10, 40, nil, 0, 0, errRequestBeyondHead},
		{false, 20, 2, 100, rpc.LatestBlockNumber, nil, 13, 20, nil},
		{false, 20, 2, 100, rpc.LatestBlockNumber, []float64{0, 10}, 31, 2, nil},
		{false, 20, 2, 100, 32, []float64{0, 10}, 31, 2, nil},
		{false, 1000, 1000, 1, rpc.PendingBlockNumber, nil, 0, 0, nil},
		{false, 1000, 1000, 2, rpc.PendingBlockNumber, nil, 32, 1, nil},
		{true, 1000, 1000, 2, rpc.PendingBlockNumber, nil, 32, 2, nil},
		{true, 1000, 1000, 2, rpc.PendingBlockNumber, []float64{0, 10}, 32, 2, nil},
		{false, 1000, 1000, 2, rpc.FinalizedBlockNumber, []float64{0, 10}, 24, 2, nil},
		{false, 1000, 1000, 2, rpc.SafeBlockNumber, []float64{0, 10}, 24, 2, nil},
	}
	for i, c := range cases {
		config := Config{
			MaxHeaderHistory: c.maxHeader,
			MaxBlockHistory:  c.maxBlock,
		}
		backend := newTestBackend(t, big.NewInt(16), big.NewInt(28), c.pending)
		oracle := NewOracle(backend, config, nil)

		first, reward, baseFee, ratio, blobBaseFee, blobRatio, err := oracle.FeeHistory(context.Background(), c.count, c.last, c.percent)
		backend.teardown()
		expReward := c.expCount
		if len(c.percent) == 0 {
			expReward = 0
		}
		expBaseFee := c.expCount
		if expBaseFee != 0 {
			expBaseFee++
		}

		if first.Uint64() != c.expFirst {
			t.Fatalf("Test case %d: first block mismatch, want %d, got %d", i, c.expFirst, first)
		}
		if len(reward) != expReward {
			t.Fatalf("Test case %d: reward array length mismatch, want %d, got %d", i, expReward, len(reward))
		}
		if len(baseFee) != expBaseFee {
			t.Fatalf("Test case %d: baseFee array length mismatch, want %d, got %d", i, expBaseFee, len(baseFee))
		}
		if len(ratio) != c.expCount {
			t.Fatalf("Test case %d: gasUsedRatio array length mismatch, want %d, got %d", i, c.expCount, len(ratio))
		}
		for _, ratio := range ratio {
			if ratio > 1 {
				t.Fatalf("Test case %d: gasUsedRatio greater than 1, got %f", i, ratio)
			}
		}
		if len(blobRatio) != c.expCount {
			t.Fatalf("Test case %d: blobGasUsedRatio array length mismatch, want %d, got %d", i, c.expCount, len(blobRatio))
		}
		for _, ratio := range blobRatio {
			if ratio > 1 {
				t.Fatalf("Test case %d: blobGasUsedRatio greater than 1, got %f", i, ratio)
			}
		}
		if len(blobBaseFee) != len(baseFee) {
			t.Fatalf("Test case %d: blobBaseFee array length mismatch, want %d, got %d", i, len(baseFee), len(blobBaseFee))
		}
		if err != c.expErr && !errors.Is(err, c.expErr) {
			t.Fatalf("Test case %d: error mismatch, want %v, got %v", i, c.expErr, err)
		}
	}
}
