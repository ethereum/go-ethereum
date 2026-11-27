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

package gasestimator

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

type dummyChainContext struct {
	consensus.ChainHeaderReader
	engine consensus.Engine
}

func (c *dummyChainContext) Engine() consensus.Engine { return c.engine }

func TestEstimateContextCancelled(t *testing.T) {
	var (
		from   = common.HexToAddress("0x1000000000000000000000000000000000000001")
		target = common.HexToAddress("0x1000000000000000000000000000000000000002")
		// Infinite loop: JUMPDEST; PUSH1 0x00; JUMP
		loopCode = []byte{byte(vm.JUMPDEST), byte(vm.PUSH1), 0x00, byte(vm.JUMP)}
	)
	statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	if err != nil {
		t.Fatalf("failed to create state db: %v", err)
	}
	statedb.SetBalance(from, uint256.NewInt(1e18), 0)
	statedb.SetCode(target, loopCode, 0)
	statedb.Finalise(params.MergedTestChainConfig.Rules(big.NewInt(1), true, 1))

	header := &types.Header{
		Number:     big.NewInt(1),
		GasLimit:   50_000_000,
		Time:       1,
		Difficulty: big.NewInt(0),
		BaseFee:    big.NewInt(1),
	}
	opts := &Options{
		Config: params.MergedTestChainConfig,
		Chain:  &dummyChainContext{engine: ethash.NewFaker()},
		Header: header,
		State:  statedb,
	}
	call := &core.Message{
		From:      from,
		To:        &target,
		GasLimit:  50_000_000,
		GasPrice:  uint256.NewInt(1),
		GasFeeCap: uint256.NewInt(1),
		GasTipCap: uint256.NewInt(1),
		Value:     uint256.NewInt(0),
	}

	t.Run("pre-cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		gas, _, err := Estimate(ctx, call, opts, 50_000_000)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got gas=%d err=%v", gas, err)
		}
	})

	t.Run("deadline-during-loop", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		defer cancel()

		gas, _, err := Estimate(ctx, call, opts, 50_000_000)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected context.DeadlineExceeded, got gas=%d err=%v", gas, err)
		}
	})
}
