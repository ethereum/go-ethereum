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

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/consensus/ethash"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/event"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rpc"
)

const testHead = 32

type testBackend struct {
	chain          *core.BlockChain
	pending        bool // pending block available
	pendingTxCount int
}

func (b *testBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	if number > testHead {
		return nil, nil
	}
	if number == rpc.EarliestBlockNumber {
		number = 0
	}
	if number == rpc.FinalizedBlockNumber {
		return b.chain.CurrentFinalBlock(), nil
	}
	if number == rpc.SafeBlockNumber {
		return b.chain.CurrentSafeBlock(), nil
	}
	if number == rpc.LatestBlockNumber {
		number = testHead
	}
	if number == rpc.PendingBlockNumber {
		if b.pending {
			number = testHead + 1
		} else {
			return nil, nil
		}
	}
	return b.chain.GetHeaderByNumber(uint64(number)), nil
}

func (b *testBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	if number > testHead {
		return nil, nil
	}
	if number == rpc.EarliestBlockNumber {
		number = 0
	}
	if number == rpc.FinalizedBlockNumber {
		number = rpc.BlockNumber(b.chain.CurrentFinalBlock().Number.Uint64())
	}
	if number == rpc.SafeBlockNumber {
		number = rpc.BlockNumber(b.chain.CurrentSafeBlock().Number.Uint64())
	}
	if number == rpc.LatestBlockNumber {
		number = testHead
	}
	if number == rpc.PendingBlockNumber {
		if b.pending {
			number = testHead + 1
		} else {
			return nil, nil
		}
	}
	return b.chain.GetBlockByNumber(uint64(number)), nil
}

func (b *testBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return b.chain.GetReceiptsByHash(hash), nil
}

func (b *testBackend) PendingBlockAndReceipts() (*types.Block, types.Receipts) {
	if b.pending {
		block := b.chain.GetBlockByNumber(testHead + 1)
		return block, b.chain.GetReceiptsByHash(block.Hash())
	}
	return nil, nil
}

func (b *testBackend) ChainConfig() *params.ChainConfig {
	return b.chain.Config()
}

func (b *testBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return nil
}

func (b *testBackend) teardown() {
	b.chain.Stop()
}

func (b *testBackend) StateAt(root common.Hash) (*state.StateDB, error) {
	return b.chain.StateAt(root)
}

func (b *testBackend) Stats() (int, int) {
	return b.pendingTxCount, 0
}

func (b *testBackend) StatsWithMinBaseFee(minBaseFee *big.Int) (int, int) {
	return b.pendingTxCount, 0
}

// newTestBackend creates a test backend. OBS: don't forget to invoke tearDown
// after use, otherwise the blockchain instance will mem-leak via goroutines.
func newTestBackend(t *testing.T, londonBlock *big.Int, pending bool, pendingTxCount int) *testBackend {
	var (
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr   = crypto.PubkeyToAddress(key.PublicKey)
		config = *params.TestChainConfig // needs copy because it is modified below
		gspec  = &core.Genesis{
			Config: &config,
			Alloc:  core.GenesisAlloc{addr: {Balance: big.NewInt(math.MaxInt64)}},
		}
		signer = types.LatestSigner(gspec.Config)
	)
	config.LondonBlock = londonBlock
	config.ArrowGlacierBlock = londonBlock
	config.GrayGlacierBlock = londonBlock

	config.ArchimedesBlock = londonBlock
	config.BernoulliBlock = londonBlock
	config.CurieBlock = londonBlock
	config.ShanghaiTime = nil
	config.DarwinTime = nil
	config.DarwinV2Time = nil
	if londonBlock != nil {
		shanghaiTime := londonBlock.Uint64() * 12
		config.ShanghaiTime = &shanghaiTime
		darwinTime := londonBlock.Uint64() * 12
		config.DarwinTime = &darwinTime
		darwinV2Time := londonBlock.Uint64() * 12
		config.DarwinV2Time = &darwinV2Time
	}

	config.TerminalTotalDifficulty = common.Big0
	engine := ethash.NewFaker()

	// Generate testing blocks
	_, blocks, _ := core.GenerateChainWithGenesis(gspec, engine, testHead+1, func(i int, b *core.BlockGen) {
		b.SetCoinbase(common.Address{1})

		var txdata types.TxData
		if londonBlock != nil && b.Number().Cmp(londonBlock) >= 0 {
			txdata = &types.DynamicFeeTx{
				ChainID:   gspec.Config.ChainID,
				Nonce:     b.TxNonce(addr),
				To:        &common.Address{},
				Gas:       30000,
				GasFeeCap: big.NewInt(100 * params.GWei),
				GasTipCap: big.NewInt(int64(i+1) * params.GWei),
				Data:      []byte{},
			}
		} else {
			txdata = &types.LegacyTx{
				Nonce:    b.TxNonce(addr),
				To:       &common.Address{},
				Gas:      21000,
				GasPrice: big.NewInt(int64(i+1) * params.GWei),
				Value:    big.NewInt(100),
				Data:     []byte{},
			}
		}
		b.AddTx(types.MustSignNewTx(key, signer, txdata))
	})
	// Construct testing chain
	chain, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), &core.CacheConfig{TrieCleanNoPrefetch: true}, gspec, nil, engine, vm.Config{}, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create local chain, %v", err)
	}
	chain.InsertChain(blocks)
	chain.SetFinalized(chain.GetBlockByNumber(25).Header())
	chain.SetSafe(chain.GetBlockByNumber(25).Header())
	return &testBackend{chain: chain, pending: pending, pendingTxCount: pendingTxCount}
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
		backend := newTestBackend(t, c.fork, false, 0)
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

func TestSuggestTipCapCongestedThreshold(t *testing.T) {
	expectedDefaultBasePricePreCurie := big.NewInt(2000)
	expectedDefaultBasePricePostCurie := big.NewInt(2)

	config := Config{
		Blocks:             3,
		Percentile:         60,
		Default:            big.NewInt(params.GWei),
		CongestedThreshold: 50,
		DefaultBasePrice:   expectedDefaultBasePricePreCurie,
	}
	var cases = []struct {
		fork      *big.Int // London fork number
		pendingTx int      // Number of pending transactions in the mempool
		expect    *big.Int // Expected gasprice suggestion
	}{
		{nil, 0, expectedDefaultBasePricePreCurie},      // No congestion - default base price
		{nil, 49, expectedDefaultBasePricePreCurie},     // No congestion - default base price
		{nil, 50, big.NewInt(params.GWei * int64(30))},  // Congestion - normal behavior
		{nil, 100, big.NewInt(params.GWei * int64(30))}, // Congestion - normal behavior

		// Fork point in genesis
		{big.NewInt(0), 0, expectedDefaultBasePricePostCurie},     // No congestion - default base price
		{big.NewInt(0), 49, expectedDefaultBasePricePostCurie},    // No congestion - default base price
		{big.NewInt(0), 50, big.NewInt(params.GWei * int64(30))},  // Congestion - normal behavior
		{big.NewInt(0), 100, big.NewInt(params.GWei * int64(30))}, // Congestion - normal behavior

		// Fork point in first block
		{big.NewInt(1), 0, expectedDefaultBasePricePostCurie},     // No congestion - default base price
		{big.NewInt(1), 49, expectedDefaultBasePricePostCurie},    // No congestion - default base price
		{big.NewInt(1), 50, big.NewInt(params.GWei * int64(30))},  // Congestion - normal behavior
		{big.NewInt(1), 100, big.NewInt(params.GWei * int64(30))}, // Congestion - normal behavior

		// Fork point in last block
		{big.NewInt(32), 0, expectedDefaultBasePricePostCurie},     // No congestion - default base price
		{big.NewInt(32), 49, expectedDefaultBasePricePostCurie},    // No congestion - default base price
		{big.NewInt(32), 50, big.NewInt(params.GWei * int64(30))},  // Congestion - normal behavior
		{big.NewInt(32), 100, big.NewInt(params.GWei * int64(30))}, // Congestion - normal behavior

		// Fork point in the future
		{big.NewInt(33), 0, expectedDefaultBasePricePreCurie},      // No congestion - default base price
		{big.NewInt(33), 49, expectedDefaultBasePricePreCurie},     // No congestion - default base price
		{big.NewInt(33), 50, big.NewInt(params.GWei * int64(30))},  // Congestion - normal behavior
		{big.NewInt(33), 100, big.NewInt(params.GWei * int64(30))}, // Congestion - normal behavior
	}
	for _, c := range cases {
		backend := newTestBackend(t, c.fork, false, c.pendingTx)
		oracle := NewOracle(backend, config)

		// The gas price sampled is: 32G, 31G, 30G, 29G, 28G, 27G
		got, err := oracle.SuggestTipCap(context.Background())
		backend.teardown()
		if err != nil {
			t.Fatalf("Failed to retrieve recommended gas price: %v", err)
		}
		if got.Cmp(c.expect) != 0 {
			t.Fatalf("Gas price mismatch, want %d, got %d", c.expect, got)
		}
	}
}
