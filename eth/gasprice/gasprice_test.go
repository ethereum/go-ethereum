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
	"math/rand"
	"testing"
	"time"

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

func newTestBackend(t *testing.T) *testBackend {
	var (
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr   = crypto.PubkeyToAddress(key.PublicKey)
		gspec  = &core.Genesis{
			Config: params.TestChainConfig,
			Alloc:  core.GenesisAlloc{addr: {Balance: big.NewInt(math.MaxInt64)}},
		}
		signer = types.LatestSigner(gspec.Config)
	)
	engine := ethash.NewFaker()
	db := rawdb.NewMemoryDatabase()
	genesis, _ := gspec.Commit(db)

	// Generate testing blocks
	blocks, _ := core.GenerateChain(params.TestChainConfig, genesis, engine, db, 32, func(i int, b *core.BlockGen) {
		b.SetCoinbase(common.Address{1})
		tx, err := types.SignTx(types.NewTransaction(b.TxNonce(addr), common.HexToAddress("deadbeef"), big.NewInt(100), 21000, big.NewInt(int64(i+1)*params.GWei), nil), signer, key)
		if err != nil {
			t.Fatalf("failed to create tx: %v", err)
		}
		b.AddTx(tx)
	})
	// Construct testing chain
	diskdb := rawdb.NewMemoryDatabase()
	gspec.Commit(diskdb)
	chain, err := core.NewBlockChain(diskdb, nil, params.TestChainConfig, engine, vm.Config{}, nil, nil)
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

func TestSuggestPrice(t *testing.T) {
	config := Config{
		Blocks:     3,
		Percentile: 60,
		Default:    big.NewInt(params.GWei),
	}
	backend := newTestBackend(t)
	oracle := NewOracle(backend, config)

	// The gas price sampled is: 32G, 31G, 30G, 29G, 28G, 27G
	got, err := oracle.SuggestPrice(context.Background())
	if err != nil {
		t.Fatalf("Failed to retrieve recommended gas price: %v", err)
	}
	expect := big.NewInt(params.GWei)
	if got.Cmp(expect) != 0 {
		t.Fatalf("Gas price mismatch, want %d, got %d", expect, got)
	}
}

func generateFakeGasPrices(min, max int) []*big.Int {
	rand.Seed(time.Now().UnixNano())
	gasPrices := make([]*big.Int, 0)
	for i := 0; i < 300; i++ {
		randGasPrice := rand.Intn(max-min+1) + min
		gasPrices = append(gasPrices, big.NewInt(int64(randGasPrice)))
	}
	return gasPrices
}

func TestRemoveLowOutliers(t *testing.T) {
	gasPrices := generateFakeGasPrices(180, 220)
	cpy := make([]*big.Int, len(gasPrices))
	copy(cpy, gasPrices)
	// add low gas prices
	cpy = append(cpy, big.NewInt(5))
	cpy = append(cpy, big.NewInt(10))
	cpy = append(cpy, big.NewInt(10))
	cpy = append(cpy, big.NewInt(10))
	cpy = append(cpy, big.NewInt(15))
	cpy = append(cpy, big.NewInt(15))
	cpy = append(cpy, big.NewInt(15))
	cpy = append(cpy, big.NewInt(15))

	res := removeOutliers(cpy)
	// It should remove all the lower  gasPrices in the extreme case
	if len(gasPrices) != len(res) {
		t.Fatalf("Low gas prices not removed, want length less than %d, got %d", len(gasPrices), len(res))
	}
}

func TestRemoveHighOutliars(t *testing.T) {
	gasPrices := generateFakeGasPrices(180, 220)
	cpy := make([]*big.Int, len(gasPrices))
	copy(cpy, gasPrices)
	// add low gas prices
	cpy = append(cpy, big.NewInt(300))
	cpy = append(cpy, big.NewInt(310))
	cpy = append(cpy, big.NewInt(350))
	cpy = append(cpy, big.NewInt(250))
	cpy = append(cpy, big.NewInt(251))
	cpy = append(cpy, big.NewInt(245))
	cpy = append(cpy, big.NewInt(255))
	cpy = append(cpy, big.NewInt(256))

	res := removeOutliers(cpy)
	// It should remove most of the higher values
	if len(cpy) < len(res) {
		t.Fatalf("Low gas prices not removed, want length less than %d, got %d", len(gasPrices), len(res))
	}
}

func TestRemoveHighAndLowOutliars(t *testing.T) {
	gasPrices := generateFakeGasPrices(180, 220)
	cpy := make([]*big.Int, len(gasPrices))
	copy(cpy, gasPrices)
	// add low gas prices
	cpy = append(cpy, big.NewInt(300))
	cpy = append(cpy, big.NewInt(310))
	cpy = append(cpy, big.NewInt(350))
	cpy = append(cpy, big.NewInt(250))
	cpy = append(cpy, big.NewInt(251))
	cpy = append(cpy, big.NewInt(245))
	cpy = append(cpy, big.NewInt(255))
	cpy = append(cpy, big.NewInt(256))
	cpy = append(cpy, big.NewInt(50))
	cpy = append(cpy, big.NewInt(100))
	cpy = append(cpy, big.NewInt(70))
	cpy = append(cpy, big.NewInt(75))
	cpy = append(cpy, big.NewInt(5))
	cpy = append(cpy, big.NewInt(0))
	cpy = append(cpy, big.NewInt(4))
	cpy = append(cpy, big.NewInt(2))
	res := removeOutliers(cpy)
	// It should remove most of the higher values
	if len(cpy) < len(res) {
		t.Fatalf("Low gas prices not removed, want length less than %d, got %d", len(gasPrices), len(res))
	}
}
