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

package eth

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// newAccessorChain generates blocks with a transfer per transaction and
// imports them into a fresh chain.
func newAccessorChain(t *testing.T, genesis *core.Genesis, engine consensus.Engine, txs int) (*Ethereum, []*types.Block) {
	t.Helper()
	signer := types.LatestSigner(genesis.Config)
	_, blocks, _ := core.GenerateChainWithGenesis(genesis, engine, 1, func(i int, b *core.BlockGen) {
		for range txs {
			tx, err := types.SignTx(types.NewTransaction(b.TxNonce(address), common.Address{1}, big.NewInt(1), params.TxGas, big.NewInt(params.InitialBaseFee), nil), signer, key)
			if err != nil {
				t.Fatal(err)
			}
			b.AddTx(tx)
		}
	})
	db := rawdb.NewMemoryDatabase()
	chain, err := core.NewBlockChain(db, genesis, engine, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(chain.Stop)
	if _, err := chain.InsertChain(blocks); err != nil {
		t.Fatal(err)
	}
	return &Ethereum{blockchain: chain, chainDb: db}, blocks
}

// The state of a transaction in the DAO fork block includes the fork's balance
// transfers, as block processing applies them.
func TestStateAtTransactionDAOFork(t *testing.T) {
	config := &params.ChainConfig{
		ChainID:        big.NewInt(1),
		HomesteadBlock: big.NewInt(0),
		DAOForkBlock:   big.NewInt(1),
		DAOForkSupport: true,
		Ethash:         new(params.EthashConfig),
	}
	drained := params.DAODrainList()[0]
	genesis := &core.Genesis{
		Config: config,
		Alloc: types.GenesisAlloc{
			address: {Balance: funds},
			drained: {Balance: big.NewInt(1)},
		},
	}
	eth, blocks := newAccessorChain(t, genesis, ethash.NewFaker(), 1)

	_, _, statedb, release, err := eth.stateAtTransaction(context.Background(), blocks[0], 0)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if have := statedb.GetBalance(params.DAORefundContract); have.Uint64() != 1 {
		t.Errorf("refund contract balance: have %v, want 1", have)
	}
	if have := statedb.GetBalance(drained); !have.IsZero() {
		t.Errorf("drained account balance: have %v, want 0", have)
	}
}

// A cancelled request does not replay the transactions before the target.
func TestStateAtTransactionCancelled(t *testing.T) {
	genesis := &core.Genesis{
		Config: &params.ChainConfig{ChainID: big.NewInt(1), HomesteadBlock: big.NewInt(0), Ethash: new(params.EthashConfig)},
		Alloc:  types.GenesisAlloc{address: {Balance: funds}},
	}
	eth, blocks := newAccessorChain(t, genesis, ethash.NewFaker(), 2)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, _, _, err := eth.stateAtTransaction(ctx, blocks[0], 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("have %v, want %v", err, context.Canceled)
	}
}
