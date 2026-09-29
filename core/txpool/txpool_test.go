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

package txpool_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/blobpool"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// Tests that the transaction pool can be created while a path-based snap sync
// is in progress, when neither the head state nor the (non-empty) genesis
// state is available.
func TestNewDuringPathSnapSync(t *testing.T) {
	gspec := &core.Genesis{
		Config: params.TestChainConfig,
		Alloc:  types.GenesisAlloc{common.Address{0x01}: {Balance: big.NewInt(1)}},
	}
	chain, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), gspec, beacon.New(ethash.NewFaker()), core.DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatalf("failed to create blockchain: %v", err)
	}
	defer chain.Stop()
	if err := chain.SnapSyncStart(); err != nil {
		t.Fatalf("failed to start snap sync: %v", err)
	}
	if _, err := chain.StateAt(chain.Genesis().Header()); err == nil {
		t.Fatal("genesis state still available during snap sync")
	}
	legacyPool := legacypool.New(legacypool.DefaultConfig, chain)
	blobPool := blobpool.New(blobpool.Config{Datadir: t.TempDir()}, chain, legacyPool.HasPendingAuth)

	pool, err := txpool.New(legacypool.DefaultConfig.PriceLimit, chain, []txpool.SubPool{legacyPool, blobPool})
	if err != nil {
		t.Fatalf("failed to create transaction pool during snap sync: %v", err)
	}
	pool.Close()
}
