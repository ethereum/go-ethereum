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

package framepool

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func TestCrossPoolRoutingAndReservations(t *testing.T) {
	config := *params.AllDevChainProtocolChanges
	config.AmsterdamTime = new(uint64)
	genesis := &core.Genesis{Config: &config, GasLimit: 30_000_000, Alloc: types.GenesisAlloc{
		poolAddress(t, 1): {Balance: big.NewInt(1e18)},
		poolAddress(t, 2): {Balance: big.NewInt(1e18)},
	}}
	chain, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), genesis, beacon.New(ethash.NewFaker()), &core.BlockChainConfig{ArchiveMode: true})
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()
	legacy := legacypool.New(legacypool.DefaultConfig, chain)
	frames := New(Config{GlobalSlots: 1, PriceBump: 10}, chain, legacy.HasPendingAuth)
	pool, err := txpool.New(1, chain, []txpool.SubPool{legacy, frames})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	add := func(tx *types.Transaction, want error) {
		t.Helper()
		if err := pool.Add([]*types.Transaction{tx}, true)[0]; !errors.Is(err, want) {
			t.Fatalf("Add: got %v, want %v", err, want)
		}
	}
	victim := poolAddress(t, 1)
	legacyTx := types.MustSignNewTx(poolKey(t, 1), types.LatestSigner(&config), &types.DynamicFeeTx{
		ChainID: config.ChainID, To: &victim, Gas: params.TxGas, GasFeeCap: big.NewInt(3e9), GasTipCap: big.NewInt(1e9),
	})
	add(legacyTx, nil)
	forged := signedPoolTx(t, 2, 0, 0, 4e9, 2e9, nil, func(tx *types.FrameTx) { tx.Sender = victim })
	add(forged, core.ErrFrameTxInvalidExecution)
	valid := signedPoolTx(t, 1, 0, 0, 4e9, 2e9, nil, nil)
	add(valid, txpool.ErrAlreadyReserved)
	if !legacy.Has(legacyTx.Hash()) || legacy.Status(legacyTx.Hash()) != txpool.TxStatusPending {
		t.Fatal("victim's pending legacy transaction displaced")
	}
	if err := legacy.ValidateTxBasics(valid); !errors.Is(err, core.ErrTxTypeNotSupported) {
		t.Fatalf("legacy accepts frame type: %v", err)
	}
	if legacy.FilterType(types.FrameTxType) {
		t.Fatal("legacy routes frame type")
	}
	candidate := signedPoolTx(t, 2, 0, 0, 3e9, 1e9, nil, nil)
	add(candidate, nil)
	if legacy.Has(candidate.Hash()) || !frames.Has(candidate.Hash()) {
		t.Fatal("frame transaction routed to wrong subpool")
	}
	pending, _ := pool.Pending(txpool.PendingFilter{})
	if txs := pending[poolAddress(t, 2)]; len(txs) != 1 || txs[0].Hash != candidate.Hash() {
		t.Fatal("frame transaction missing from aggregate Pending")
	}
	// The victim's higher-priced frame would evict the sole candidate if reservation
	// acquisition happened after eviction. Both pools must remain untouched.
	add(valid, txpool.ErrAlreadyReserved)
	if !frames.Has(candidate.Hash()) || frames.Has(valid.Hash()) || !legacy.Has(legacyTx.Hash()) {
		t.Fatal("failed reservation mutated pool contents")
	}
	// A successful legacy replacement proves the victim is still owned by legacy.
	replacement := types.MustSignNewTx(poolKey(t, 1), types.LatestSigner(&config), &types.DynamicFeeTx{
		ChainID: config.ChainID, To: &victim, Gas: params.TxGas, GasFeeCap: big.NewInt(4e9), GasTipCap: big.NewInt(2e9),
	})
	add(replacement, nil)
	if !legacy.Has(replacement.Hash()) || legacy.Has(legacyTx.Hash()) {
		t.Fatal("legacy reservation lost after frame rejection")
	}
	// Below-basefee frame transactions stay admissible and retrievable, but
	// cannot be selected by the block producer at the current base fee.
	frames.Clear()
	belowBaseFee := signedPoolTx(t, 2, 0, 0, 10, 2, nil, nil)
	add(belowBaseFee, nil)
	pending, _ = pool.Pending(txpool.PendingFilter{BaseFee: uint256.MustFromBig(chain.CurrentBlock().BaseFee)})
	if len(pending[poolAddress(t, 2)]) != 0 || pool.Get(belowBaseFee.Hash()) == nil {
		t.Fatal("below-basefee admission or producer filtering incorrect")
	}
	if pool.Get(candidate.Hash()) != nil || pool.GetRLP(candidate.Hash(), 72) != nil || pool.GetMetadata(candidate.Hash()) != nil {
		t.Fatal("removed frame transaction remains exposed to network retrieval")
	}
	if pool.GetMetadata(belowBaseFee.Hash()) == nil || len(pool.GetRLP(belowBaseFee.Hash(), 72)) == 0 {
		t.Fatal("live frame transaction missing from network retrieval")
	}
}
