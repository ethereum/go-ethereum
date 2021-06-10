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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
)

func TestFeeHistory(t *testing.T) {
	var cases = []struct {
		pending             bool
		maxHeader, maxBlock int
		count               int
		last                rpc.BlockNumber
		percent             []float64
		expFirst            rpc.BlockNumber
		expCount            int
		expErr              error
	}{
		{false, 0, 0, 10, 30, nil, 21, 10, nil},
		{false, 0, 0, 10, 30, []float64{0, 10}, 21, 10, nil},
		{false, 0, 0, 10, 30, []float64{20, 10}, 0, 0, errInvalidPercentiles},
		{false, 0, 0, 1000000000, 30, nil, 0, 31, nil},
		{false, 0, 0, 1000000000, rpc.LatestBlockNumber, nil, 0, 33, nil},
		{false, 0, 0, 10, 40, nil, 31, 2, nil},
		{true, 0, 0, 10, 40, nil, 31, 2, nil},
		{false, 0, 0, 10, 400, nil, 0, 0, nil},
		{false, 20, 2, 100, rpc.LatestBlockNumber, nil, 13, 20, nil},
		{false, 20, 2, 100, rpc.LatestBlockNumber, []float64{0, 10}, 31, 2, nil},
		{false, 20, 2, 100, 100, []float64{0, 10}, 31, 2, nil},
		{false, 0, 0, 1, rpc.PendingBlockNumber, nil, 0, 0, nil},
		{false, 0, 0, 2, rpc.PendingBlockNumber, nil, 32, 1, nil},
		{true, 0, 0, 2, rpc.PendingBlockNumber, nil, 32, 2, nil},
	}
	for _, c := range cases {
		config := Config{
			MaxHeaderHistory: c.maxHeader,
			MaxBlockHistory:  c.maxBlock,
		}
		backend := newTestBackend(t, big.NewInt(16), c.pending)
		oracle := NewOracle(backend, config)

		first, reward, baseFee, ratio, err := oracle.FeeHistory(context.Background(), c.count, c.last, c.percent)

		expReward := c.expCount
		if len(c.percent) == 0 {
			expReward = 0
		}
		expBaseFee := c.expCount
		if expBaseFee != 0 {
			expBaseFee++
		}

		if first != c.expFirst {
			t.Fatalf("First block mismatch, want %d, got %d", c.expFirst, first)
		}
		if len(reward) != expReward {
			t.Fatalf("Reward array length mismatch, want %d, got %d", expReward, len(reward))
		}
		if len(baseFee) != expBaseFee {
			t.Fatalf("BaseFee array length mismatch, want %d, got %d", expBaseFee, len(baseFee))
		}
		if len(ratio) != c.expCount {
			t.Fatalf("GasUsedRatio array length mismatch, want %d, got %d", c.expCount, len(ratio))
		}
		if err != c.expErr {
			t.Fatalf("Error mismatch, want %v, got %v", c.expErr, err)
		}
	}
}
