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

package miner

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func TestCommitTransactionsReturnsTimeoutOnCancelledEVM(t *testing.T) {
	var (
		key, _   = crypto.GenerateKey()
		from     = crypto.PubkeyToAddress(key.PublicKey)
		contract = common.HexToAddress("0x1000000000000000000000000000000000000000")
		header   = &types.Header{
			Number:     big.NewInt(1),
			GasLimit:   1_000_000,
			GasUsed:    0,
			BaseFee:    big.NewInt(params.InitialBaseFee),
			Difficulty: big.NewInt(0),
			Coinbase:   common.HexToAddress("0x2000000000000000000000000000000000000000"),
		}
	)
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	statedb.CreateAccount(from)
	statedb.AddBalance(from, uint256.MustFromBig(new(big.Int).Lsh(big.NewInt(1), 100)), tracing.BalanceIncreaseGenesisBalance)
	statedb.CreateAccount(contract)
	// Store a value before entering a cancelled jump loop. Without the miner-side
	// Cancelled check, the interpreter clears its stop token and this write survives.
	statedb.SetCode(contract, common.Hex2Bytes("600160005560075b600756"), tracing.CodeChangeUnspecified)
	statedb.Finalise(true)

	evm := vm.NewEVM(core.NewEVMBlockContext(header, nil, &header.Coinbase), statedb, params.TestChainConfig, vm.Config{})
	defer evm.Release()
	evm.Cancel()

	signer := types.MakeSigner(params.TestChainConfig, header.Number, header.Time)
	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		Nonce:    0,
		To:       &contract,
		Gas:      100_000,
		GasPrice: big.NewInt(params.InitialBaseFee),
	})
	plainTxs := newTransactionsByPriceAndNonce(signer, map[common.Address][]*txpool.LazyTransaction{
		from: {{
			Tx:        tx,
			Hash:      tx.Hash(),
			Time:      time.Now(),
			GasFeeCap: uint256.MustFromBig(tx.GasFeeCap()),
			GasTipCap: uint256.MustFromBig(tx.GasTipCap()),
			Gas:       tx.Gas(),
		}},
	}, header.BaseFee)
	blobTxs := newTransactionsByPriceAndNonce(signer, map[common.Address][]*txpool.LazyTransaction{}, header.BaseFee)
	env := &environment{
		signer:  signer,
		state:   statedb,
		gasPool: core.NewGasPool(header.GasLimit),
		header:  header,
		evm:     evm,
	}
	miner := &Miner{chainConfig: params.TestChainConfig}

	err := miner.commitTransactions(context.Background(), env, plainTxs, blobTxs, nil)
	if !errors.Is(err, errBlockInterruptedByTimeout) {
		t.Fatalf("unexpected error: got %v, want %v", err, errBlockInterruptedByTimeout)
	}
	if len(env.txs) != 0 {
		t.Fatalf("interrupted transaction included: %d txs", len(env.txs))
	}
	if got := statedb.GetState(contract, common.Hash{}); got != (common.Hash{}) {
		t.Fatalf("interrupted transaction state was not reverted: %x", got)
	}
	if got := env.gasPool.Used(); got != 0 {
		t.Fatalf("interrupted transaction gas was not restored: %d", got)
	}
	if header.GasUsed != 0 {
		t.Fatalf("interrupted transaction updated header gas: %d", header.GasUsed)
	}
}
