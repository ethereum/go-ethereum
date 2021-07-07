// Copyright 2020 The go-ethereum Authors
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

package gasprice

import (
	"context"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

type testBackend struct {
	chain *core.BlockChain
}

func (b *testBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	if number == rpc.LatestBlockNumber {
		return b.chain.CurrentBlock().Header(), nil
	}
	return b.chain.GetHeaderByNumber(uint64(number)), nil
}

func (b *testBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	if number == rpc.LatestBlockNumber {
		return b.chain.CurrentBlock(), nil
	}
	return b.chain.GetBlockByNumber(uint64(number)), nil
}

func (b *testBackend) ChainConfig() *params.ChainConfig {
	return b.chain.Config()
}

func newTestBackend(t *testing.T, londonBlock *big.Int) *testBackend {
	var (
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr   = crypto.PubkeyToAddress(key.PublicKey)
		gspec  = &core.Genesis{
			Config: params.TestChainConfig,
			Alloc:  core.GenesisAlloc{addr: {Balance: big.NewInt(math.MaxInt64)}},
		}
		signer = types.LatestSigner(gspec.Config)
	)
	if londonBlock != nil {
		gspec.Config.LondonBlock = londonBlock
		signer = types.LatestSigner(gspec.Config)
	} else {
		gspec.Config.LondonBlock = nil
	}
	engine := ethash.NewFaker()
	db := rawdb.NewMemoryDatabase()
	genesis, _ := gspec.Commit(db)

	// Generate testing blocks
	blocks, _ := core.GenerateChain(gspec.Config, genesis, engine, db, 32, func(i int, b *core.BlockGen) {
		b.SetCoinbase(common.Address{1})

		var tx *types.Transaction
		if londonBlock != nil && b.Number().Cmp(londonBlock) >= 0 {
			txdata := &types.DynamicFeeTx{
				ChainID:   gspec.Config.ChainID,
				Nonce:     b.TxNonce(addr),
				To:        &common.Address{},
				Gas:       30000,
				GasFeeCap: big.NewInt(100 * params.GWei),
				GasTipCap: big.NewInt(int64(i+1) * params.GWei),
				Data:      []byte{},
			}
			tx = types.NewTx(txdata)
		} else {
			txdata := &types.LegacyTx{
				Nonce:    b.TxNonce(addr),
				To:       &common.Address{},
				Gas:      21000,
				GasPrice: big.NewInt(int64(i+1) * params.GWei),
				Value:    big.NewInt(100),
				Data:     []byte{},
			}
			tx = types.NewTx(txdata)
		}
		tx, err := types.SignTx(tx, signer, key)
		if err != nil {
			t.Fatalf("failed to create tx: %v", err)
		}
		b.AddTx(tx)
	})
	// Construct testing chain
	diskdb := rawdb.NewMemoryDatabase()
	gspec.Commit(diskdb)
	chain, err := core.NewBlockChain(diskdb, nil, gspec.Config, engine, vm.Config{}, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create local chain, %v", err)
	}
	chain.InsertChain(blocks)
	return &testBackend{chain: chain}
}

func (b *testBackend) CurrentHeader() *types.Header {
	return b.chain.CurrentHeader()
}

func (b *testBackend) GetBlockByNumber(number uint64) *types.Block {
	return b.chain.GetBlockByNumber(number)
}

func TestSuggestTipCap(t *testing.T) {
	config := Config{
		Blocks:     3,
		Percentile: 60,
		Default:    big.NewInt(params.GWei),
	}
	var cases = []struct {
		fork   *big.Int // London fork number
		expect *big.Int // Expected gasprice suggestion
	}{
		{nil, big.NewInt(params.GWei * int64(30))},
		{big.NewInt(0), big.NewInt(params.GWei * int64(30))},  // Fork point in genesis
		{big.NewInt(1), big.NewInt(params.GWei * int64(30))},  // Fork point in first block
		{big.NewInt(32), big.NewInt(params.GWei * int64(30))}, // Fork point in last block
		{big.NewInt(33), big.NewInt(params.GWei * int64(30))}, // Fork point in the future
	}
	for _, c := range cases {
		backend := newTestBackend(t, c.fork)
		oracle := NewOracle(backend, config)

		// The gas price sampled is: 32G, 31G, 30G, 29G, 28G, 27G
		got, err := oracle.SuggestTipCap(context.Background())
		if err != nil {
			t.Fatalf("Failed to retrieve recommended gas price: %v", err)
		}
		if got.Cmp(c.expect) != 0 {
			t.Fatalf("Gas price mismatch, want %d, got %d", c.expect, got)
		}
	}
}
