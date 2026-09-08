// Copyright 2026 The go-ethereum Authors
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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/ethapi/override"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

// TestMakeHeadersPostMergeDifficultyOverride checks that a non-zero difficulty
// override on a post-merge chain is a no-op. Applying it would leave Random
// unset in NewEVMBlockContext, so Rules become pre-merge while EIP-2935 still
// runs and ProcessParentBlockHash panics on PUSH0.
func TestMakeHeadersPostMergeDifficultyOverride(t *testing.T) {
	t.Parallel()

	sim := &simulator{
		base: &types.Header{
			Number:     big.NewInt(10),
			Time:       50,
			Difficulty: big.NewInt(0),
			GasLimit:   30_000_000,
			MixDigest:  common.Hash{0x42},
		},
		chainConfig: params.MergedTestChainConfig,
	}
	number := (*hexutil.Big)(big.NewInt(11))
	timestamp := hexutil.Uint64(62)
	difficulty := (*hexutil.Big)(big.NewInt(1))
	headers, err := sim.makeHeaders([]simBlock{{
		BlockOverrides: &override.BlockOverrides{
			Number:     number,
			Time:       &timestamp,
			Difficulty: difficulty,
		},
	}})
	if err != nil {
		t.Fatalf("makeHeaders: %v", err)
	}
	if len(headers) != 1 {
		t.Fatalf("expected 1 header, got %d", len(headers))
	}
	header := headers[0]
	if header.Difficulty == nil || header.Difficulty.Sign() != 0 {
		t.Fatalf("post-merge difficulty override should be a no-op, got %v", header.Difficulty)
	}
	// NewEVMBlockContext sets Random only when Difficulty == 0. Rules uses
	// Random != nil as the merge flag, so a non-zero difficulty would disable
	// Shanghai/Prague (including PUSH0) while chainConfig.IsPrague stays true.
	isMerge := header.Difficulty.Sign() == 0
	rules := sim.chainConfig.Rules(header.Number, isMerge, header.Time)
	if !rules.IsMerge || !rules.IsShanghai || !rules.IsPrague {
		t.Fatalf("post-merge difficulty override must not disable merge rules (merge=%v shanghai=%v prague=%v difficulty=%s)",
			rules.IsMerge, rules.IsShanghai, rules.IsPrague, header.Difficulty)
	}
}

// TestSimulateV1PostMergeDifficultyOverride is the crash regression for
// eth_simulateV1 with blockOverrides.difficulty on a Prague chain.
func TestSimulateV1PostMergeDifficultyOverride(t *testing.T) {
	t.Parallel()

	var (
		sender    = common.Address{0xc0}
		recipient = common.Address{0xc1}
		gspec     = &core.Genesis{
			Config: params.MergedTestChainConfig,
			Alloc: types.GenesisAlloc{
				sender: {Balance: big.NewInt(params.Ether)},
			},
		}
	)
	backend := newTestBackend(t, 0, gspec, beacon.New(ethash.NewFaker()), func(i int, b *core.BlockGen) {})
	api := NewBlockChainAPI(backend)
	latest := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
	difficulty := (*hexutil.Big)(big.NewInt(1))
	opts := simOpts{
		BlockStateCalls: []simBlock{{
			BlockOverrides: &override.BlockOverrides{
				Difficulty: difficulty,
			},
			Calls: []TransactionArgs{{
				From:  &sender,
				To:    &recipient,
				Value: (*hexutil.Big)(big.NewInt(1000)),
			}},
		}},
	}
	results, err := api.SimulateV1(context.Background(), opts, &latest)
	if err != nil {
		t.Fatalf("post-merge difficulty override should be a no-op, got error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 simulated block, got %d", len(results))
	}
	if results[0].Block.Difficulty().Sign() != 0 {
		t.Fatalf("post-merge simulated block difficulty should remain 0, got %s", results[0].Block.Difficulty())
	}
	if len(results[0].Calls) != 1 {
		t.Fatalf("expected 1 call result, got %d", len(results[0].Calls))
	}
	if results[0].Calls[0].Status != 1 {
		t.Fatalf("call should succeed, status=%d error=%v", results[0].Calls[0].Status, results[0].Calls[0].Error)
	}
}
