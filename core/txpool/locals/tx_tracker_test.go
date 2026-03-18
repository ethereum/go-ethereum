// Copyright 2025 The go-ethereum Authors
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

package locals

import (
	"math/big"
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/ethash"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"github.com/XinFinOrg/XDPoSChain/core/txpool"
	"github.com/XinFinOrg/XDPoSChain/core/txpool/legacypool"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/params"
)

var (
	key, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	address = crypto.PubkeyToAddress(key.PublicKey)
	funds   = big.NewInt(1000000000000000)
	gspec   = &core.Genesis{
		Config: params.TestChainConfig,
		Alloc: types.GenesisAlloc{
			address: {Balance: funds},
		},
		BaseFee: big.NewInt(params.InitialBaseFee),
	}
	signer = types.LatestSigner(gspec.Config)
)

type testEnv struct {
	chain   *core.BlockChain
	pool    *txpool.TxPool
	tracker *TxTracker
	genDb   ethdb.Database
}

func newTestEnv(t *testing.T, n int, gasTip uint64, journal string) *testEnv {
	genDb, blocks, _ := core.GenerateChainWithGenesis(gspec, ethash.NewFaker(), n, func(i int, gen *core.BlockGen) {
		tx, err := types.SignTx(types.NewTransaction(gen.TxNonce(address), common.Address{0x00}, big.NewInt(1000), params.TxGas, gen.BaseFee(), nil), signer, key)
		if err != nil {
			panic(err)
		}
		gen.AddTx(tx)
	})

	db := rawdb.NewMemoryDatabase()
	chain, _ := core.NewBlockChain(db, nil, gspec, ethash.NewFaker(), vm.Config{})

	legacyPool := legacypool.New(legacypool.DefaultConfig, chain)
	pool, err := txpool.New(gasTip, chain, []txpool.SubPool{legacyPool})
	if err != nil {
		t.Fatalf("Failed to create tx pool: %v", err)
	}
	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("Failed to process block %d: %v", n, err)
	}
	if err := pool.Sync(); err != nil {
		t.Fatalf("Failed to sync the txpool, %v", err)
	}
	return &testEnv{
		chain:   chain,
		pool:    pool,
		tracker: New(journal, time.Minute, gspec.Config, pool),
		genDb:   genDb,
	}
}

func (env *testEnv) close() {
	env.chain.Stop()
}

func (env *testEnv) setGasTip(gasTip uint64) {
	env.pool.SetGasTip(new(big.Int).SetUint64(gasTip))
}

func (env *testEnv) makeTx(nonce uint64, gasPrice *big.Int) *types.Transaction {
	if nonce == 0 {
		head := env.chain.CurrentHeader()
		state, _ := env.chain.StateAt(head.Root)
		nonce = state.GetNonce(address)
	}
	if gasPrice == nil {
		gasPrice = big.NewInt(params.GWei)
	}
	tx, _ := types.SignTx(types.NewTransaction(nonce, common.Address{0x00}, big.NewInt(1000), params.TxGas, gasPrice, nil), signer, key)
	return tx
}

func (env *testEnv) commit() {
	head := env.chain.CurrentBlock()
	block := env.chain.GetBlock(head.Hash(), head.Number.Uint64())
	blocks, _ := core.GenerateChain(env.chain.Config(), block, ethash.NewFaker(), env.genDb, 1, func(i int, gen *core.BlockGen) {
		tx, err := types.SignTx(types.NewTransaction(gen.TxNonce(address), common.Address{0x00}, big.NewInt(1000), params.TxGas, gen.BaseFee(), nil), signer, key)
		if err != nil {
			panic(err)
		}
		gen.AddTx(tx)
	})
	env.chain.InsertChain(blocks)
	if err := env.pool.Sync(); err != nil {
		panic(err)
	}
}
