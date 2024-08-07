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
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
)

const testHead = 32

type testBackend struct {
	chain   *core.BlockChain
	pending bool // pending block available
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

func (b *testBackend) Pending() (*types.Block, types.Receipts, *state.StateDB) {
	if b.pending {
		block := b.chain.GetBlockByNumber(testHead + 1)
		state, _ := b.chain.StateAt(block.Root())
		return block, b.chain.GetReceiptsByHash(block.Hash()), state
	}
	return nil, nil, nil
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

// newTestBackend creates a test backend. OBS: don't forget to invoke tearDown
// after use, otherwise the blockchain instance will mem-leak via goroutines.
func newTestBackend(t *testing.T, londonBlock *big.Int, cancunBlock *big.Int, pending bool) *testBackend {
	if londonBlock != nil && cancunBlock != nil && londonBlock.Cmp(cancunBlock) == 1 {
		panic("cannot define test backend with cancun before london")
	}
	var (
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr   = crypto.PubkeyToAddress(key.PublicKey)
		config = *params.TestChainConfig // needs copy because it is modified below
		gspec  = &core.Genesis{
			Config: &config,
			Alloc:  types.GenesisAlloc{addr: {Balance: big.NewInt(math.MaxInt64)}},
		}
		signer = types.LatestSigner(gspec.Config)

		// Compute empty blob hash.
		emptyBlob          = kzg4844.Blob{}
		emptyBlobCommit, _ = kzg4844.BlobToCommitment(&emptyBlob)
		emptyBlobVHash     = kzg4844.CalcBlobHashV1(sha256.New(), &emptyBlobCommit)
	)
	config.LondonBlock = londonBlock
	config.ArrowGlacierBlock = londonBlock
	config.GrayGlacierBlock = londonBlock
	var engine consensus.Engine = beacon.New(ethash.NewFaker())
	td := params.GenesisDifficulty.Uint64()

	if cancunBlock != nil {
		ts := gspec.Timestamp + cancunBlock.Uint64()*10 // fixed 10 sec block time in blockgen
		config.ShanghaiTime = &ts
		config.CancunTime = &ts
		signer = types.LatestSigner(gspec.Config)
	}

	// Generate testing blocks
	db, blocks, _ := core.GenerateChainWithGenesis(gspec, engine, testHead+1, func(i int, b *core.BlockGen) {
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

		if cancunBlock != nil && b.Number().Cmp(cancunBlock) >= 0 {
			b.SetPoS()

			// put more blobs in each new block
			for j := 0; j < i && j < 6; j++ {
				blobTx := &types.BlobTx{
					ChainID:    uint256.MustFromBig(gspec.Config.ChainID),
					Nonce:      b.TxNonce(addr),
					To:         common.Address{},
					Gas:        30000,
					GasFeeCap:  uint256.NewInt(100 * params.GWei),
					GasTipCap:  uint256.NewInt(uint64(i+1) * params.GWei),
					Data:       []byte{},
					BlobFeeCap: uint256.NewInt(1),
					BlobHashes: []common.Hash{emptyBlobVHash},
					Value:      uint256.NewInt(100),
					Sidecar:    nil,
				}
				b.AddTx(types.MustSignNewTx(key, signer, blobTx))
			}
		}
		td += b.Difficulty().Uint64()
	})
	// Construct testing chain
	gspec.Config.TerminalTotalDifficulty = new(big.Int).SetUint64(td)
	chain, err := core.NewBlockChain(db, &core.CacheConfig{TrieCleanNoPrefetch: true}, gspec, nil, engine, vm.Config{}, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create local chain, %v", err)
	}
	if i, err := chain.InsertChain(blocks); err != nil {
		panic(fmt.Errorf("error inserting block %d: %w", i, err))
	}
	chain.SetFinalized(chain.GetBlockByNumber(25).Header())
	chain.SetSafe(chain.GetBlockByNumber(25).Header())

	return &testBackend{chain: chain, pending: pending}
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
		backend := newTestBackend(t, c.fork, nil, false)
		oracle := NewOracle(backend, config, big.NewInt(params.GWei))

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
