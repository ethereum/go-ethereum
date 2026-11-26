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

package downloader

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/params"
)

// Tests that a head state recoverable from the state history keeps full sync.
func TestSyncModeKeepsFullSyncWithRecoverableHeadState(t *testing.T) {
	var (
		key, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		address = crypto.PubkeyToAddress(key.PublicKey)
		gspec   = &core.Genesis{
			Config:  params.TestChainConfig,
			Alloc:   types.GenesisAlloc{address: {Balance: big.NewInt(1_000_000_000_000_000_000)}},
			BaseFee: big.NewInt(params.InitialBaseFee),
		}
		signer   = types.LatestSigner(gspec.Config)
		engine   = ethash.NewFaker()
		imported = 10
		pending  = int(state.TriesInMemory) + 10
	)
	_, blocks, _ := core.GenerateChainWithGenesis(gspec, engine, imported+pending, func(i int, gen *core.BlockGen) {
		tx, err := types.SignTx(types.NewTransaction(gen.TxNonce(address), common.Address{0x01}, big.NewInt(1), params.TxGas, gen.BaseFee(), nil), signer, key)
		if err != nil {
			t.Fatal(err)
		}
		gen.AddTx(tx)
	})
	db, err := rawdb.Open(rawdb.NewMemoryDatabase(), rawdb.OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	chain, err := core.NewBlockChain(db, gspec, engine, core.DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()

	if n, err := chain.InsertChain(blocks[:imported]); err != nil {
		t.Fatalf("block %d: failed to import: %v", n, err)
	}
	moder := newSyncModer(ethconfig.FullSync, chain, db)
	if mode := moder.get(false); mode != ethconfig.FullSync {
		t.Fatalf("sync mode before the payloads: have %v, want %v", mode, ethconfig.FullSync)
	}
	for _, block := range blocks[imported:] {
		if _, err := chain.InsertBlockWithoutSetHead(context.Background(), block, false); err != nil {
			t.Fatalf("block %d: failed to import without setting the head: %v", block.NumberU64(), err)
		}
	}
	head := chain.CurrentBlock()
	if head.Hash() != blocks[imported-1].Hash() {
		t.Fatalf("head moved: have %d, want %d", head.Number, imported)
	}
	if chain.HasState(head.Root) {
		t.Fatal("head state still in memory; the payloads did not push it below the disk layer")
	}
	if !chain.StateRecoverable(head.Root) {
		t.Fatal("head state not recoverable from the state history")
	}
	if mode := moder.get(false); mode != ethconfig.FullSync {
		t.Fatalf("sync mode after the payloads: have %v, want %v", mode, ethconfig.FullSync)
	}
	tip := blocks[len(blocks)-1]
	if _, err := chain.SetCanonical(tip); err != nil {
		t.Fatalf("failed to set the head to the last payload: %v", err)
	}
	if chain.CurrentBlock().Hash() != tip.Hash() || !chain.HasState(tip.Root()) {
		t.Fatalf("head after the forkchoice update: have %d, want %d with state", chain.CurrentBlock().Number, tip.NumberU64())
	}
}
