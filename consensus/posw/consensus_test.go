// Copyright 2026 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package posw

import (
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// testChain rejects canonical-head lookups so engine tests must use parent hashes.
type testChain struct {
	config      *params.ChainConfig
	genesis     *types.Header
	headers     map[common.Hash]*types.Header
	headerReads int
	stakes      map[common.Hash]*uint256.Int
	stakeErr    error
	stakeParent common.Hash
}

func (c *testChain) Config() *params.ChainConfig  { return c.config }
func (c *testChain) CurrentHeader() *types.Header { panic("must use the candidate's parent") }
func (c *testChain) GetHeader(hash common.Hash, number uint64) *types.Header {
	c.headerReads++
	h := c.headers[hash]
	if h != nil && h.Number.Uint64() == number {
		return h
	}
	return nil
}
func (c *testChain) GetHeaderByNumber(uint64) *types.Header         { panic("must use the candidate's fork") }
func (c *testChain) GetHeaderByHash(hash common.Hash) *types.Header { return c.headers[hash] }
func (c *testChain) GetBlock(common.Hash, uint64) *types.Block {
	panic("staking difficulty must use headers")
}
func (c *testChain) GetShardLockedStakes(parent *types.Header, _ common.Address) (*uint256.Int, error) {
	c.stakeParent = parent.Hash()
	if c.stakeErr != nil {
		return nil, c.stakeErr
	}
	return c.stakes[parent.Hash()], nil
}

func newTestChain() (*testChain, *types.Header) {
	config := *params.AllEthashProtocolChanges
	config.QuarkChain = params.NewTestQuarkChainConfig(0)
	config.QuarkChain.PoSW.MinimumDifficulty = big.NewInt(100)
	parent := &types.Header{Number: new(big.Int), Difficulty: new(big.Int), GasLimit: 12_000_000,
		BaseFee: big.NewInt(params.InitialBaseFee), UncleHash: types.EmptyUncleHash,
		MixDigest: common.BigToHash(big.NewInt(100)), Extra: (&types.QKCExtra{XShardGasLimit: 12_000_000 / params.XShardGasLimitDivisor}).Encode()}
	return &testChain{config: &config, genesis: parent, headers: map[common.Hash]*types.Header{parent.Hash(): parent}, stakes: make(map[common.Hash]*uint256.Int)}, parent
}

func newTestEngine(t *testing.T, chain *testChain) *PoSW {
	t.Helper()
	engine, err := New(chain.config.QuarkChain, chain.genesis)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { engine.Close() })
	return engine
}

func child(parent *types.Header) *types.Header {
	h := types.CopyHeader(parent)
	h.ParentHash, h.Number, h.Time = parent.Hash(), new(big.Int).Add(parent.Number, common.Big1), parent.Time+1
	h.BaseFee = new(big.Int).Mul(parent.BaseFee, big.NewInt(7))
	h.BaseFee.Div(h.BaseFee, big.NewInt(8))
	return h
}

// TestNew checks that missing or invalid shard configurations and genesis headers
// fail at construction.
func TestNew(t *testing.T) {
	chain, genesis := newTestChain()
	invalid := params.NewTestQuarkChainConfig(0)
	invalid.PoSW.AdjustmentCutoff = 0
	for _, config := range []*params.QuarkChainConfig{nil, {}, invalid} {
		if engine, err := New(config, genesis); err == nil || engine != nil {
			t.Fatalf("invalid config accepted: engine %v, error %v", engine, err)
		}
	}
	for _, genesis := range []*types.Header{nil, {GasLimit: params.MaxGasLimit + 1}} {
		if engine, err := New(chain.config.QuarkChain, genesis); err == nil || engine != nil {
			t.Fatalf("invalid genesis accepted: engine %v, error %v", engine, err)
		}
	}
}

// TestBatchHeaderResults checks independent, ordered results: an invalid parent
// is reported even when its child's header checks pass, as in Ethash. A parent
// with a missing number or invalid gas limit must not make its child's
// verification panic.
func TestBatchHeaderResults(t *testing.T) {
	chain, parent := newTestChain()
	engine := newTestEngine(t, chain)
	first := child(parent)
	first.Nonce = types.EncodeNonce(1)
	second := child(first)
	second.Nonce = types.BlockNonce{}
	abort, results := engine.VerifyHeaders(chain, []*types.Header{first, second})
	defer close(abort)
	if err := <-results; err == nil {
		t.Fatal("malformed parent accepted")
	}
	if err := <-results; err != nil {
		t.Fatalf("otherwise valid child header: %v", err)
	}
	t.Run("missing parent number", func(t *testing.T) {
		first.Number = nil
		second.ParentHash = first.Hash()
		abort, results := engine.VerifyHeaders(chain, []*types.Header{first, second})
		defer close(abort)
		for range 2 {
			if err := <-results; !errors.Is(err, consensus.ErrInvalidNumber) {
				t.Fatalf("missing parent number: got %v, want ErrInvalidNumber", err)
			}
		}
	})
	t.Run("zero parent gas target", func(t *testing.T) {
		first := child(parent)
		first.GasLimit, first.GasUsed = 0, 1
		second := child(first)
		second.GasLimit, second.GasUsed = parent.GasLimit, 0
		abort, results := engine.VerifyHeaders(chain, []*types.Header{first, second})
		defer close(abort)
		for range 2 {
			if err := <-results; err == nil {
				t.Fatal("malformed parent gas fields accepted")
			}
		}
	})
}

// TestHeaderVerification checks batch parent lookup, shared field validation,
// shard configuration, difficulty and timestamps without reading staking state.
func TestHeaderVerification(t *testing.T) {
	chain, parent := newTestChain()
	engine := newTestEngine(t, chain)
	if err := engine.VerifyHeader(chain, parent); !errors.Is(err, consensus.ErrInvalidNumber) {
		t.Fatalf("known genesis: got %v, want ErrInvalidNumber", err)
	}
	genesisAbort, genesisResult := engine.VerifyHeaders(chain, []*types.Header{parent})
	defer close(genesisAbort)
	if err := <-genesisResult; !errors.Is(err, consensus.ErrInvalidNumber) {
		t.Fatalf("batch genesis: got %v, want ErrInvalidNumber", err)
	}
	first := child(parent)
	second := child(first)
	// Neither child is stored, and staking reads must never run in this phase.
	chain.stakeErr = errors.New("state unavailable")
	abort, results := engine.VerifyHeaders(chain, []*types.Header{first, second})
	defer close(abort)
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if chain.stakeParent != (common.Hash{}) {
		t.Fatal("header validation read staking state")
	}
	for _, change := range []struct {
		name   string
		mutate func(*types.Header)
	}{
		{"extra", func(h *types.Header) { h.Extra = nil }},
		{"shard", func(h *types.Header) { e, _ := h.QKCExtra(); e.ShardID = 1; h.Extra = e.Encode() }},
		{"base difficulty", func(h *types.Header) { h.MixDigest = common.BigToHash(big.NewInt(99)) }},
		{"time", func(h *types.Header) { h.Time = parent.Time }},
		{"future", func(h *types.Header) { h.Time = uint64(time.Now().Unix()) + 60 }},
		{"number", func(h *types.Header) { h.Number.SetInt64(-1) }},
		{"gas limit increase", func(h *types.Header) {
			h.GasLimit += 2
			e, _ := h.QKCExtra()
			e.XShardGasLimit = h.GasLimit / params.XShardGasLimitDivisor
			h.Extra = e.Encode()
		}},
		{"gas limit decrease", func(h *types.Header) {
			h.GasLimit -= 2
			e, _ := h.QKCExtra()
			e.XShardGasLimit = h.GasLimit / params.XShardGasLimitDivisor
			h.Extra = e.Encode()
		}},
		{"cross-shard gas limit", func(h *types.Header) { e, _ := h.QKCExtra(); e.XShardGasLimit--; h.Extra = e.Encode() }},
		{"gas used", func(h *types.Header) { h.GasUsed = h.GasLimit + 1 }},
		{"missing base fee", func(h *types.Header) { h.BaseFee = nil }},
		{"base fee", func(h *types.Header) { h.BaseFee.Add(h.BaseFee, common.Big1) }},
	} {
		t.Run(change.name, func(t *testing.T) {
			h := types.CopyHeader(first)
			change.mutate(h)
			if err := engine.VerifyHeader(chain, h); err == nil {
				t.Fatal("invalid header accepted")
			}
		})
	}
}

// TestFixedGasLimitAcrossLondon checks that BaseFee activation does not change
// the genesis gas limit, and that pre-London headers still omit BaseFee.
func TestFixedGasLimitAcrossLondon(t *testing.T) {
	for _, london := range []int64{1, 2} {
		t.Run(big.NewInt(london).String(), func(t *testing.T) {
			chain, parent := newTestChain()
			chain.config.LondonBlock = big.NewInt(london)
			header := child(parent)
			parent.BaseFee = nil
			header.ParentHash = parent.Hash()
			chain.headers = map[common.Hash]*types.Header{parent.Hash(): parent}
			if chain.config.IsLondon(header.Number) {
				header.BaseFee = eip1559.CalcBaseFee(chain.config, parent)
			} else {
				header.BaseFee = nil
			}
			engine := newTestEngine(t, chain)
			if err := engine.VerifyHeader(chain, header); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestHeaderForkFields checks field presence before and at each execution fork,
// QKC's empty/zero constraints, and acceptance of nonempty execution commitments.
func TestHeaderForkFields(t *testing.T) {
	chain, parent := newTestChain()
	chain.config.ShanghaiTime = uint64Ptr(10)
	chain.config.CancunTime = uint64Ptr(20)
	chain.config.PragueTime, chain.config.OsakaTime = uint64Ptr(30), uint64Ptr(30)
	chain.config.AmsterdamTime = uint64Ptr(40)
	engine := newTestEngine(t, chain)
	headerAt := func(timestamp uint64) *types.Header {
		header := child(parent)
		header.Time = timestamp
		if timestamp >= 10 {
			header.WithdrawalsHash = &types.EmptyWithdrawalsHash
		}
		if timestamp >= 20 {
			header.ParentBeaconRoot = new(common.Hash)
			header.BlobGasUsed, header.ExcessBlobGas = new(uint64), new(uint64)
		}
		if timestamp >= 30 {
			header.RequestsHash = &common.Hash{1}
		}
		if timestamp >= 40 {
			header.BlockAccessListHash, header.SlotNumber = &common.Hash{2}, new(uint64)
		}
		return header
	}
	for _, field := range []struct {
		name   string
		fork   uint64
		toggle func(*types.Header)
	}{
		{"withdrawals", 10, func(h *types.Header) {
			if h.WithdrawalsHash == nil {
				h.WithdrawalsHash = &types.EmptyWithdrawalsHash
			} else {
				h.WithdrawalsHash = nil
			}
		}},
		{"beacon root", 20, func(h *types.Header) {
			if h.ParentBeaconRoot == nil {
				h.ParentBeaconRoot = new(common.Hash)
			} else {
				h.ParentBeaconRoot = nil
			}
		}},
		{"blob gas used", 20, func(h *types.Header) {
			if h.BlobGasUsed == nil {
				h.BlobGasUsed = new(uint64)
			} else {
				h.BlobGasUsed = nil
			}
		}},
		{"excess blob gas", 20, func(h *types.Header) {
			if h.ExcessBlobGas == nil {
				h.ExcessBlobGas = new(uint64)
			} else {
				h.ExcessBlobGas = nil
			}
		}},
		{"requests", 30, func(h *types.Header) {
			if h.RequestsHash == nil {
				h.RequestsHash = &types.EmptyRequestsHash
			} else {
				h.RequestsHash = nil
			}
		}},
		{"block access list", 40, func(h *types.Header) {
			if h.BlockAccessListHash == nil {
				h.BlockAccessListHash = &types.EmptyBlockAccessListHash
			} else {
				h.BlockAccessListHash = nil
			}
		}},
		{"slot number", 40, func(h *types.Header) {
			if h.SlotNumber == nil {
				h.SlotNumber = new(uint64)
			} else {
				h.SlotNumber = nil
			}
		}},
	} {
		t.Run(field.name, func(t *testing.T) {
			for _, timestamp := range []uint64{field.fork - 1, field.fork} {
				header := headerAt(timestamp)
				if err := engine.VerifyHeader(chain, header); err != nil {
					t.Fatalf("valid header at %d: %v", timestamp, err)
				}
				field.toggle(header)
				if err := engine.VerifyHeader(chain, header); err == nil {
					t.Fatalf("invalid field presence accepted at %d", timestamp)
				}
			}
		})
	}
	for _, field := range []struct {
		name   string
		mutate func(*types.Header)
	}{
		{"withdrawals", func(h *types.Header) { h.WithdrawalsHash = &common.Hash{1} }},
		{"beacon root", func(h *types.Header) { h.ParentBeaconRoot = &common.Hash{1} }},
		{"blob gas used", func(h *types.Header) { *h.BlobGasUsed = 1 }},
		{"excess blob gas", func(h *types.Header) { *h.ExcessBlobGas = 1 }},
		{"slot number", func(h *types.Header) { *h.SlotNumber = 1 }},
	} {
		t.Run("nonempty "+field.name, func(t *testing.T) {
			header := headerAt(40)
			field.mutate(header)
			if err := engine.VerifyHeader(chain, header); err == nil {
				t.Fatal("nonempty reserved field accepted")
			}
		})
	}
}

func uint64Ptr(value uint64) *uint64 { return &value }

// TestPrepareAndFinalize checks base difficulty initialization, clearing stale
// seals while preserving other QKC fields, the fixed genesis gas limit, invalid
// inputs and subsidy crediting.
func TestPrepareAndFinalize(t *testing.T) {
	chain, parent := newTestChain()
	config := chain.config.QuarkChain.PoSW
	config.CoinbaseAmount, config.RewardTaxRate = big.NewInt(101), big.NewRat(1, 2)
	engine := newTestEngine(t, chain)
	h := child(parent)
	h.GasLimit += 2
	h.MixDigest = common.Hash{}
	h.Difficulty.SetInt64(1)
	h.Nonce = types.EncodeNonce(1)
	extra, _ := h.QKCExtra()
	extra.PrevRootBlock, extra.Cursor.RootBlockHeight = common.Hash{1}, 2
	wantExtra := *extra
	extra.Nonce, extra.MixHash = types.EncodeNonce(3), common.Hash{4}
	h.Extra = extra.Encode()
	if err := engine.Prepare(chain, h); err != nil {
		t.Fatal(err)
	}
	if h.QKCDifficulty().Int64() != 100 || h.Difficulty.Sign() != 0 || h.Nonce != (types.BlockNonce{}) {
		t.Fatal("incorrect prepared difficulty fields")
	}
	if h.GasLimit != parent.GasLimit {
		t.Fatalf("prepared gas limit: have %d, want %d", h.GasLimit, parent.GasLimit)
	}
	if got, _ := h.QKCExtra(); *got != wantExtra {
		t.Fatalf("prepared extra: got %+v, want %+v", *got, wantExtra)
	}
	if err := engine.VerifyHeader(chain, h); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []struct {
		name  string
		extra []byte
	}{
		{"extra", nil},
		{"version", (&types.QKCExtra{Version: 1}).Encode()},
		{"shard", (&types.QKCExtra{ShardID: 1}).Encode()},
	} {
		t.Run(invalid.name, func(t *testing.T) {
			header := types.CopyHeader(h)
			header.Extra = invalid.extra
			if err := engine.Prepare(chain, header); err == nil {
				t.Fatal("prepared invalid QKCExtra")
			}
		})
	}
	h.Time = parent.Time
	if err := engine.Prepare(chain, h); err == nil {
		t.Fatal("accepted non-increasing time")
	}
	db, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	if err != nil {
		t.Fatal(err)
	}
	engine.Finalize(chain, h, db, &types.Body{})
	if db.GetBalance(h.Coinbase).Uint64() != 50 {
		t.Fatal("configured subsidy not credited")
	}
}

// TestSealParentFork checks real Ethash seals against the parent-state staking
// discount, fork-specific window counts and missing state or ancestors.
// Disabled staking must still verify PoW without reading stakes or ancestors.
func TestSealParentFork(t *testing.T) {
	chain, genesis := newTestChain()
	c := chain.config.QuarkChain.PoSW
	c.StakingEnabled, c.WindowSize, c.DiffDivider, c.StakePerBlock = true, 3, 100, big.NewInt(100)
	engine := newTestEngine(t, chain)
	parent := child(genesis)
	parent.Coinbase = common.Address{1}
	chain.headers[parent.Hash()] = parent
	chain.stakes[parent.Hash()] = uint256.NewInt(100)
	h := child(parent)
	h.Coinbase = common.Address{2}
	// A real Ethash seal at difficulty one qualifies only through PoSW.
	// A nonzero QKC nonce also catches accidental use of geth's reserved Nonce.
	const nonce = uint64(0x0102030405060708)
	extra, _ := h.QKCExtra()
	extra.Nonce = types.EncodeNonce(nonce)
	h.Extra = extra.Encode()
	hash, _ := h.QKCSealHash()
	cache := engine.cache(h.Number.Uint64())
	digest, result := hashimotoLight(datasetSize(h.Number.Uint64()), cache, hash[:], nonce)
	if new(big.Int).SetBytes(result).Cmp(new(big.Int).Div(new(big.Int).Lsh(common.Big1, 256), big.NewInt(100))) <= 0 {
		t.Fatal("fixture also satisfies base difficulty")
	}
	extra.MixHash = common.BytesToHash(digest)
	h.Extra = extra.Encode()
	if err := engine.VerifySeal(chain, h); err != nil {
		t.Fatal(err)
	}
	if chain.stakeParent != parent.Hash() {
		t.Fatal("read the wrong parent state")
	}
	c.StakingEnabled = false
	t.Run("ordinary PoW without staking or history", func(t *testing.T) {
		powChain := *chain
		powChain.headers = nil
		powChain.headerReads = 0
		powChain.stakeErr = errors.New("staking must not be read")
		powChain.stakeParent = common.Hash{}
		// Expose only the header interface: disabled staking needs no StakingReader.
		powReader := struct{ consensus.ChainHeaderReader }{&powChain}
		if err := engine.VerifySeal(powReader, h); err == nil || err.Error() != "invalid Ethash proof of work" {
			t.Fatalf("ordinary PoW with insufficient work: %v", err)
		}
		// Difficulty one accepts this real seal even without staking or ancestors.
		header := types.CopyHeader(h)
		header.MixDigest = common.BigToHash(common.Big1)
		hash, _ := header.QKCSealHash()
		digest, _ := hashimotoLight(datasetSize(header.Number.Uint64()), cache, hash[:], nonce)
		extra, _ := header.QKCExtra()
		extra.MixHash = common.BytesToHash(digest)
		header.Extra = extra.Encode()
		if err := engine.VerifySeal(powReader, header); err != nil {
			t.Fatalf("ordinary PoW with valid seal: %v", err)
		}
		if powChain.stakeParent != (common.Hash{}) || powChain.headerReads != 0 {
			t.Fatal("ordinary PoW read staking state or ancestors")
		}
	})
	c.StakingEnabled = true
	chain.stakeErr = consensus.ErrPrunedAncestor
	if err := engine.VerifySeal(chain, h); !errors.Is(err, consensus.ErrPrunedAncestor) {
		t.Fatalf("missing state: %v", err)
	}
	chain.stakeErr = nil
	extra.MixHash = common.Hash{}
	h.Extra = extra.Encode()
	if err := engine.VerifySeal(chain, h); err == nil {
		t.Fatal("invalid mix accepted")
	}
	// The same coinbase on this parent has already exhausted its one-block quota.
	h.Coinbase = parent.Coinbase
	if got, err := GetAdjustedDifficulty(c, chain, h); err != nil || got.Int64() != 100 {
		t.Fatalf("fork quota: %v %v", got, err)
	}
	t.Run("window boundary and fork", func(t *testing.T) {
		chain, genesis := newTestChain()
		c := chain.config.QuarkChain.PoSW
		c.StakingEnabled, c.WindowSize, c.DiffDivider, c.StakePerBlock = true, 3, 100, big.NewInt(100)
		// Only the two ancestors inside the window are available.
		older := child(genesis)
		older.Number, older.ParentHash, older.Coinbase = big.NewInt(2), common.Hash{99}, common.Address{1}
		parent := child(older)
		chain.headers[older.Hash()], chain.headers[parent.Hash()] = older, parent
		chain.stakes[parent.Hash()] = uint256.NewInt(200)
		header := child(parent)
		if got, err := GetAdjustedDifficulty(c, chain, header); err != nil || got.Int64() != 100 {
			t.Fatalf("two-block quota: %v %v", got, err)
		}
		fork := types.CopyHeader(parent)
		fork.Coinbase = common.Address{2}
		chain.headers[fork.Hash()], chain.stakes[fork.Hash()] = fork, uint256.NewInt(200)
		header.ParentHash = fork.Hash()
		if got, err := GetAdjustedDifficulty(c, chain, header); err != nil || got.Int64() != 1 {
			t.Fatalf("side-fork quota: %v %v", got, err)
		}
		delete(chain.headers, older.Hash())
		if _, err := GetAdjustedDifficulty(c, chain, header); !errors.Is(err, consensus.ErrUnknownAncestor) {
			t.Fatalf("missing ancestor inside window: %v", err)
		}
		c.WindowSize = 1
		if got, err := GetAdjustedDifficulty(c, chain, header); err != nil || got.Int64() != 1 {
			t.Fatalf("empty history window: %v %v", got, err)
		}
	})
}
