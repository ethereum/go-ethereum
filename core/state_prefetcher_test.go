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

package core

import (
	"math/big"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// TestPrefetchOrder checks that heavy transactions are promoted to the front
// of the prefetch queue, heaviest first, while the rest keeps block order
// regardless of gas.
func TestPrefetchOrder(t *testing.T) {
	gas := []uint64{21000, 5000000, 300000, 1200000, 21000}
	txs := make(types.Transactions, len(gas))
	for i, g := range gas {
		txs[i] = types.NewTx(&types.LegacyTx{Gas: g})
	}
	want := []int{1, 3, 0, 2, 4}
	if have := prefetchOrder(txs); !slices.Equal(have, want) {
		t.Fatalf("wrong prefetch order, have %v want %v", have, want)
	}
	// Equally heavy transactions keep block order.
	gas = []uint64{21000, 2000000, 2000000, 21000}
	txs = make(types.Transactions, len(gas))
	for i, g := range gas {
		txs[i] = types.NewTx(&types.LegacyTx{Gas: g})
	}
	want = []int{1, 2, 0, 3}
	if have := prefetchOrder(txs); !slices.Equal(have, want) {
		t.Fatalf("wrong prefetch order, have %v want %v", have, want)
	}
}

// TestPrefetchCaughtUpSkip checks that prefetch workers skip transactions the
// main pass has already reached.
func TestPrefetchCaughtUpSkip(t *testing.T) {
	gspec := &Genesis{
		Config: params.TestChainConfig,
		Alloc:  types.GenesisAlloc{benchRootAddr: {Balance: benchRootFunds}},
	}
	_, blocks, _ := GenerateChainWithGenesis(gspec, ethash.NewFaker(), 1, func(i int, gen *BlockGen) {
		for j := 0; j < 8; j++ {
			input := make([]byte, 96)
			input[31] = 1 // G1 generator x
			input[63] = 2 // G1 generator y
			input[95] = byte(j + 1)

			to := common.BytesToAddress([]byte{7})
			tx, err := types.SignNewTx(benchRootKey, gen.Signer(), &types.LegacyTx{
				Nonce:    gen.TxNonce(benchRootAddr),
				To:       &to,
				Gas:      100000,
				Data:     input,
				GasPrice: gen.header.BaseFee,
			})
			if err != nil {
				panic(err)
			}
			gen.AddTx(tx)
		}
	})
	chain, _ := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, ethash.NewFaker(), nil)
	defer chain.Stop()

	// Count top level call frames, one per executed transaction.
	var executed atomic.Int64
	cfg := vm.Config{Tracer: &tracing.Hooks{
		OnEnter: func(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
			if depth == 0 {
				executed.Add(1)
			}
		},
	}}
	block := blocks[0]
	numTxs := int64(len(block.Transactions()))

	run := func(mainIndex int64) int64 {
		statedb, err := chain.StateAt(chain.GetHeaderByNumber(0))
		if err != nil {
			t.Fatal(err)
		}
		var execIndex atomic.Int64
		execIndex.Store(mainIndex)
		executed.Store(0)
		chain.prefetcher.Prefetch(block, statedb, nil, nil, cfg, nil, &execIndex)
		return executed.Load()
	}
	if have := run(-1); have != numTxs {
		t.Errorf("nothing executed by main pass: prefetched %d txs, want %d", have, numTxs)
	}
	if have := run(numTxs - 1); have != 0 {
		t.Errorf("main pass fully caught up: prefetched %d txs, want 0", have)
	}
	if have := run(2); have != numTxs-3 {
		t.Errorf("main pass at tx 2: prefetched %d txs, want %d", have, numTxs-3)
	}
}
