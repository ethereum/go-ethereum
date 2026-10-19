// Copyright 2026 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package tracetest

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestCallTracerDeposit(t *testing.T) {
	for _, fork := range []string{"pre-amsterdam", "amsterdam"} {
		for _, tracerName := range []string{"callTracer", "callTracerLegacy"} {
			t.Run(fork+"/"+tracerName, func(t *testing.T) {
				config := *params.MergedTestChainConfig
				config.QuarkChain = &params.QuarkChainConfig{ShardID: 1, ShardIDs: []uint32{0, 1}}
				if fork == "amsterdam" {
					config.AmsterdamTime = new(uint64)
				}
				var (
					from     = common.HexToAddress("0xa1")
					to       = common.HexToAddress("0xe0a")
					coinbase = common.HexToAddress("0xc0ffee")
					header   = &types.Header{Number: big.NewInt(1), Difficulty: new(big.Int), BaseFee: big.NewInt(1), GasLimit: 30_000_000}
					tx       = types.NewTx(&types.DepositTx{
						From: from, To: &to, Value: uint256.NewInt(1000), GasPrice: uint256.NewInt(2),
						Gas: params.XShardSendGas, IsMove: true,
					})
				)
				sdb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
				require.NoError(t, err)
				// A move credits this contract without running its storage write.
				sdb.SetCode(to, program.New().Op(vm.CALLVALUE).Push(0).Op(vm.SSTORE).Bytes(), tracing.CodeChangeUnspecified)
				sdb.Finalise(config.Rules(header.Number, true, header.Time))
				sdb.SetTxContext(tx.Hash(), 0, 1)

				tracer, err := tracers.DefaultDirectory.New(tracerName, new(tracers.Context), json.RawMessage(`{"withLog":true}`), &config)
				require.NoError(t, err)
				logState := state.NewHookedState(sdb, tracer.Hooks)
				evm := vm.NewEVM(core.NewEVMBlockContext(header, nil, &coinbase), logState, &config, vm.Config{Tracer: tracer.Hooks})
				defer evm.Release()
				msg, err := core.TransactionToMessage(tx, types.LatestSigner(&config), header.BaseFee)
				require.NoError(t, err)
				receipt, _, err := core.ApplyTransactionWithEVM(context.Background(), msg, core.NewGasPool(header.GasLimit), sdb, header.Number, common.Hash{}, header.Time, tx, evm)
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
				require.Equal(t, params.XShardSendGas, receipt.GasUsed)
				require.Equal(t, uint256.NewInt(1000), sdb.GetBalance(to))
				require.Equal(t, common.Hash{}, sdb.GetState(to, common.Hash{}))

				result, err := tracer.GetResult()
				require.NoError(t, err)
				var trace callTrace
				require.NoError(t, json.Unmarshal(result, &trace))
				require.Equal(t, "CALL", trace.Type)
				require.Equal(t, from, trace.From)
				require.Equal(t, &to, trace.To)
				require.NotNil(t, trace.Value)
				require.Equal(t, tx.Value(), trace.Value.ToInt())
				require.NotNil(t, trace.Gas)
				require.Equal(t, tx.Gas(), uint64(*trace.Gas))
				require.NotNil(t, trace.GasUsed)
				require.Equal(t, receipt.GasUsed, uint64(*trace.GasUsed))
				require.Empty(t, trace.Error)
				require.Empty(t, trace.Calls)
				require.Empty(t, trace.Logs)
			})
		}
	}
}
