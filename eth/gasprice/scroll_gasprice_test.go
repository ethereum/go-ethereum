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
	"math/big"
	"math/rand"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/event"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/scroll-tech/go-ethereum/trie"
)

var (
	blockGasLimit             = params.TxGas * 3
	maxTxPayloadBytesPerBlock = 790
)

type testTxData struct {
	priorityFee int64
	gasLimit    uint64
	payloadSize uint64
}

type scrollTestBackend struct {
	block      *types.Block
	receipts   []*types.Receipt
	curieBlock *big.Int
}

func (b *scrollTestBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	if number == rpc.LatestBlockNumber || number == rpc.BlockNumber(b.block.NumberU64()) {
		return b.block.Header(), nil
	}
	return nil, nil
}

func (b *scrollTestBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	return b.block, nil
}

func (b *scrollTestBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return b.receipts, nil
}

func (b *scrollTestBackend) Pending() (*types.Block, types.Receipts, *state.StateDB) {
	panic("not implemented")
}

func (b *scrollTestBackend) ChainConfig() *params.ChainConfig {
	config := params.TestChainConfig
	config.Scroll.MaxTxPayloadBytesPerBlock = &maxTxPayloadBytesPerBlock
	config.CurieBlock = b.curieBlock
	return config
}

func (b *scrollTestBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return nil
}

func (b *scrollTestBackend) PendingBlockAndReceipts() (*types.Block, types.Receipts) {
	return nil, nil
}

func (b *scrollTestBackend) StateAt(root common.Hash) (*state.StateDB, error) {
	return nil, nil
}

var _ OracleBackend = (*scrollTestBackend)(nil)

func GenerateRandomBytes(length int) []byte {
	b := make([]byte, length)
	for i := range b {
		b[i] = byte(rand.Intn(256))
	}
	return b
}

func newScrollTestBackend(_ *testing.T, txs []testTxData, curieBlock *big.Int) *scrollTestBackend {
	var (
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		signer = types.LatestSigner(params.TestChainConfig)
	)
	// only the most recent block is considered for optimism priority fee suggestions, so this is
	// where we add the test transactions
	ts := []*types.Transaction{}
	rs := []*types.Receipt{}
	header := types.Header{}
	header.GasLimit = blockGasLimit
	var nonce uint64
	for _, tx := range txs {
		txdata := &types.DynamicFeeTx{
			ChainID:   params.TestChainConfig.ChainID,
			Nonce:     nonce,
			To:        &common.Address{},
			Gas:       params.TxGas,
			GasFeeCap: big.NewInt(100 * params.GWei),
			GasTipCap: big.NewInt(tx.priorityFee),
			Data:      GenerateRandomBytes(int(tx.payloadSize)),
		}
		t := types.MustSignNewTx(key, signer, txdata)
		ts = append(ts, t)
		r := types.Receipt{}
		r.GasUsed = tx.gasLimit
		header.GasUsed += r.GasUsed
		rs = append(rs, &r)
		nonce++
	}
	hasher := trie.NewStackTrie(nil)
	b := types.NewBlock(&header, ts, nil, nil, hasher)
	return &scrollTestBackend{block: b, receipts: rs, curieBlock: curieBlock}
}

func TestSuggestScrollPriorityFee(t *testing.T) {
	expectedDefaultBasePricePreCurie := big.NewInt(20000)
	expectedDefaultBasePricePostCurie := big.NewInt(100)
	cases := []struct {
		curieBlock *big.Int
		txdata     []testTxData
		want       *big.Int
	}{

		// Pre-Curie block gas limit test cases
		{
			// block gas limit well under capacity, expect min priority fee suggestion
			curieBlock: big.NewInt(1),
			txdata:     []testTxData{{params.GWei, 21000, 0}},
			want:       expectedDefaultBasePricePreCurie,
		},
		{
			// 2 txs, gas limit still under capacity, expect min priority fee suggestion
			curieBlock: big.NewInt(1),
			txdata:     []testTxData{{params.GWei, 21000, 0}, {params.GWei, 21000, 0}},
			want:       expectedDefaultBasePricePreCurie,
		},
		{
			// 2 txs w same priority fee (1 gwei), but second tx puts it right over gas limit capacity
			curieBlock: big.NewInt(1),
			txdata:     []testTxData{{params.GWei, 21000, 0}, {params.GWei, 21001, 0}},
			want:       big.NewInt(1100000000), // 10 percent over 1 gwei, the median
		},
		{
			// 3 txs, full block. return 10% over the median tx (10 gwei * 10% == 11 gwei)
			curieBlock: big.NewInt(1),
			txdata:     []testTxData{{10 * params.GWei, 21000, 0}, {1 * params.GWei, 21000, 0}, {100 * params.GWei, 21000, 0}},
			want:       big.NewInt(11 * params.GWei),
		},

		// Pre-Curie block payload size test cases
		{
			// block block payload well under capacity, expect min priority fee suggestion
			curieBlock: big.NewInt(1),
			txdata:     []testTxData{{params.GWei, 0, 139}},
			want:       expectedDefaultBasePricePreCurie,
		},
		{
			// 2 txs, still under block payload capacity, expect min priority fee suggestion
			curieBlock: big.NewInt(1),
			txdata:     []testTxData{{params.GWei, 0, 139}, {params.GWei, 0, 139}},
			want:       expectedDefaultBasePricePreCurie,
		},
		{
			// 2 txs w same priority fee (1 gwei), but second tx puts it right over block payload capacity
			curieBlock: big.NewInt(1),
			txdata:     []testTxData{{params.GWei, 0, 139}, {params.GWei, 0, 140}},
			want:       big.NewInt(1100000000), // 10 percent over 1 gwei, the median
		},
		{
			// 3 txs, full block. return 10% over the median tx (10 gwei * 10% == 11 gwei)
			curieBlock: big.NewInt(1),
			txdata:     []testTxData{{20 * params.GWei, 0, 139}, {1 * params.GWei, 0, 140}, {100 * params.GWei, 0, 140}},
			want:       big.NewInt(22 * params.GWei),
		},

		// Post Curie block gas limit test cases
		{
			// block gas limit well under capacity, expect min priority fee suggestion
			curieBlock: big.NewInt(0),
			txdata:     []testTxData{{params.GWei, 21000, 0}},
			want:       expectedDefaultBasePricePostCurie,
		},
		{
			// 2 txs, gas limit still under capacity, expect min priority fee suggestion
			curieBlock: big.NewInt(0),
			txdata:     []testTxData{{params.GWei, 21000, 0}, {params.GWei, 21000, 0}},
			want:       expectedDefaultBasePricePostCurie,
		},
		{
			// 2 txs w same priority fee (1 gwei), but second tx puts it right over gas limit capacity
			curieBlock: big.NewInt(0),
			txdata:     []testTxData{{params.GWei, 21000, 0}, {params.GWei, 21001, 0}},
			want:       big.NewInt(1100000000), // 10 percent over 1 gwei, the median
		},
		{
			// 3 txs, full block. return 10% over the median tx (10 gwei * 10% == 11 gwei)
			curieBlock: big.NewInt(0),
			txdata:     []testTxData{{10 * params.GWei, 21000, 0}, {1 * params.GWei, 21000, 0}, {100 * params.GWei, 21000, 0}},
			want:       big.NewInt(11 * params.GWei),
		},

		// Post Curie block payload size test cases
		{
			// block block payload well under capacity, expect min priority fee suggestion
			curieBlock: big.NewInt(0),
			txdata:     []testTxData{{params.GWei, 0, 139}},
			want:       expectedDefaultBasePricePostCurie,
		},
		{
			// 2 txs, still under block payload capacity, expect min priority fee suggestion
			curieBlock: big.NewInt(0),
			txdata:     []testTxData{{params.GWei, 0, 139}, {params.GWei, 0, 139}},
			want:       expectedDefaultBasePricePostCurie,
		},
		{
			// 2 txs w same priority fee (1 gwei), but second tx puts it right over block payload capacity
			curieBlock: big.NewInt(0),
			txdata:     []testTxData{{params.GWei, 0, 139}, {params.GWei, 0, 140}},
			want:       big.NewInt(1100000000), // 10 percent over 1 gwei, the median
		},
		{
			// 3 txs, full block. return 10% over the median tx (10 gwei * 10% == 11 gwei)
			curieBlock: big.NewInt(0),
			txdata:     []testTxData{{20 * params.GWei, 0, 139}, {1 * params.GWei, 0, 140}, {100 * params.GWei, 0, 140}},
			want:       big.NewInt(22 * params.GWei),
		},
	}
	for i, c := range cases {
		backend := newScrollTestBackend(t, c.txdata, c.curieBlock)
		oracle := NewOracle(backend, Config{DefaultBasePrice: expectedDefaultBasePricePreCurie, DefaultGasTipCap: expectedDefaultBasePricePostCurie})
		got := oracle.SuggestScrollPriorityFee(context.Background(), backend.block.Header())
		if got.Cmp(c.want) != 0 {
			t.Errorf("Gas price mismatch for test case %d: want %d, got %d", i, c.want, got)
		}
	}
}

// Benchmark API QPS for gas price oracle with different transaction patterns
func BenchmarkScrollGasPriceAPIQPS(b *testing.B) {
	// Create diverse transaction patterns to fill blocks
	createTransactionSet := func(blockIndex int, txCount int) []testTxData {
		txs := make([]testTxData, txCount)
		for i := 0; i < txCount; i++ {
			// Create varied transactions to avoid cache hits on tx.Size()
			txs[i] = testTxData{
				priorityFee: int64((blockIndex*txCount + i + 1) * params.GWei), // Unique priority fees
				gasLimit:    uint64(21000 + (i%10)*1000),                       // Varied gas limits
				payloadSize: uint64(i%10) * 256,                                // Varied payload sizes: 0, 256, 512, 768, 1024, 1280, 1536, 1792, 2048, 2304 bytes
			}
		}
		return txs
	}

	testScenarios := []struct {
		name        string
		curieBlock  *big.Int
		blocksCount int
		txsPerBlock int
		description string
	}{
		{
			name:        "LightLoad_PreCurie",
			curieBlock:  big.NewInt(100), // After current block
			blocksCount: 1,
			txsPerBlock: 5,
			description: "Light load with 5 txs, pre-Curie",
		},
		{
			name:        "LightLoad_PostCurie",
			curieBlock:  big.NewInt(0), // Curie from genesis
			blocksCount: 1,
			txsPerBlock: 5,
			description: "Light load with 5 txs, post-Curie",
		},
		{
			name:        "MediumLoad_PreCurie",
			curieBlock:  big.NewInt(100),
			blocksCount: 3,
			txsPerBlock: 20,
			description: "Medium load with 20 txs per block, 3 blocks",
		},
		{
			name:        "MediumLoad_PostCurie",
			curieBlock:  big.NewInt(0),
			blocksCount: 3,
			txsPerBlock: 20,
			description: "Medium load with 20 txs per block, 3 blocks",
		},
		{
			name:        "HighLoad_PostCurie",
			curieBlock:  big.NewInt(0),
			blocksCount: 1,
			txsPerBlock: 100,
			description: "High load with 100 txs, testing congestion scenarios",
		},
	}

	for _, scenario := range testScenarios {
		b.Run(scenario.name, func(b *testing.B) {
			// Create backend with diverse transactions
			txs := createTransactionSet(0, scenario.txsPerBlock)
			backend := newScrollTestBackend(nil, txs, scenario.curieBlock)

			config := Config{
				DefaultBasePrice: big.NewInt(20000),
				DefaultGasTipCap: big.NewInt(100),
				MaxPrice:         big.NewInt(500 * params.GWei),
			}
			oracle := NewOracle(backend, config)
			ctx := context.Background()

			// Sub-benchmarks for different API methods
			b.Run("SuggestScrollPriorityFee", func(b *testing.B) {
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						_ = oracle.SuggestScrollPriorityFee(ctx, backend.block.Header())
					}
				})
			})

			b.Run("SuggestTipCap", func(b *testing.B) {
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						_, _ = oracle.SuggestTipCap(ctx)
					}
				})
			})

			b.Run("calculateSuggestPriorityFee", func(b *testing.B) {
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						_, _ = oracle.calculateSuggestPriorityFee(ctx, backend.block.Header())
					}
				})
			})
		})
	}
}
