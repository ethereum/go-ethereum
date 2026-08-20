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

package core

import (
	"context"
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// hookRecorder counts every live-tracer hook invocation, keyed by hook name.
type hookRecorder struct {
	mu     sync.Mutex
	counts map[string]int
}

func (r *hookRecorder) mark(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.counts == nil {
		r.counts = make(map[string]int)
	}
	r.counts[name]++
}

func (r *hookRecorder) total() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int
	for _, c := range r.counts {
		n += c
	}
	return n
}

func (r *hookRecorder) fired() map[string]int {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]int, len(r.counts))
	for k, v := range r.counts {
		out[k] = v
	}
	return out
}

// hooks builds a live tracer covering the hooks a plain value-transfer block
// exercises, both at block scope and inside the EVM.
func (r *hookRecorder) hooks() *tracing.Hooks {
	return &tracing.Hooks{
		OnBlockStart: func(tracing.BlockEvent) { r.mark("OnBlockStart") },
		OnBlockEnd:   func(error) { r.mark("OnBlockEnd") },
		OnTxStart: func(*tracing.VMContext, *types.Transaction, common.Address) {
			r.mark("OnTxStart")
		},
		OnTxEnd: func(*types.Receipt, error) { r.mark("OnTxEnd") },
		OnEnter: func(int, byte, common.Address, common.Address, []byte, uint64, *big.Int) {
			r.mark("OnEnter")
		},
		OnExit: func(int, []byte, uint64, error, bool) { r.mark("OnExit") },
		OnOpcode: func(uint64, byte, uint64, uint64, tracing.OpContext, []byte, int, error) {
			r.mark("OnOpcode")
		},
		OnBalanceChange: func(common.Address, *big.Int, *big.Int, tracing.BalanceChangeReason) {
			r.mark("OnBalanceChange")
		},
		OnStorageChange: func(common.Address, common.Hash, common.Hash, common.Hash) {
			r.mark("OnStorageChange")
		},
		OnLog: func(*types.Log) { r.mark("OnLog") },
	}
}

// TestProcessBlockTracerOptIn asserts that ExecuteConfig.EnableTracer is the
// single switch governing the node-wide live tracer, at every layer.
//
// The tracer held in BlockChainConfig.VmConfig is a stateful singleton shared by
// the whole node, and its hooks are only safe to drive from the chain-insertion
// goroutine. Callers that execute blocks elsewhere — debug_executionWitness runs
// ProcessBlock straight off an RPC goroutine — leave EnableTracer false and must
// see no hook of any kind fire. Gating only the OnBlockStart/OnBlockEnd envelope
// is not enough: the tx-level hooks reach the same singleton through the
// vm.Config handed to the processor, and firing them without an enclosing block
// is worse than not gating at all.
func TestProcessBlockTracerOptIn(t *testing.T) {
	var (
		engine = beacon.New(ethash.NewFaker())
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr   = crypto.PubkeyToAddress(key.PublicKey)
		to     = common.HexToAddress("0x00000000000000000000000000000000deadbeef")
		funds  = new(big.Int).Mul(big.NewInt(1000), big.NewInt(params.Ether))
		config = *params.AllEthashProtocolChanges
		gspec  = &Genesis{
			Config:  &config,
			BaseFee: big.NewInt(params.InitialBaseFee),
			Alloc:   types.GenesisAlloc{addr: {Balance: funds}},
		}
	)
	gspec.Config.TerminalTotalDifficulty = common.Big0
	gspec.Config.ShanghaiTime = u64(0)
	signer := types.LatestSigner(gspec.Config)

	// A single block carrying one value transfer, so that both the block-scope
	// and the tx-scope hooks have something to report.
	_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		tx, err := types.SignNewTx(key, signer, &types.DynamicFeeTx{
			ChainID:   gspec.Config.ChainID,
			Nonce:     0,
			To:        &to,
			Gas:       100000,
			GasFeeCap: big.NewInt(params.InitialBaseFee * 2),
			GasTipCap: big.NewInt(1),
			Value:     big.NewInt(1000),
		})
		if err != nil {
			t.Fatalf("failed to sign tx: %v", err)
		}
		b.AddTx(tx)
	})
	if got := len(blocks[0].Transactions()); got != 1 {
		t.Fatalf("test block carries %d transactions, want 1", got)
	}

	// newChain returns a chain whose node-wide vm.Config carries the live tracer,
	// as `geth --vmtrace` configures it.
	newChain := func(t *testing.T) (*BlockChain, *hookRecorder) {
		t.Helper()
		recorder := new(hookRecorder)
		options := DefaultConfig().WithStateScheme(rawdb.HashScheme)
		options.VmConfig = vm.Config{Tracer: recorder.hooks()}

		chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, engine, options)
		if err != nil {
			t.Fatalf("failed to create chain: %v", err)
		}
		t.Cleanup(chain.Stop)
		recorder.counts = nil // discard OnBlockchainInit/OnGenesisBlock
		return chain, recorder
	}

	parentRoot := gspec.ToBlock().Root()

	t.Run("opted out", func(t *testing.T) {
		// The exact config debug_executionWitness uses.
		chain, recorder := newChain(t)
		_, err := chain.ProcessBlock(context.Background(), parentRoot, blocks[0], ExecuteConfig{
			WriteState:   false,
			EnableTracer: false,
			MakeWitness:  true,
		})
		if err != nil {
			t.Fatalf("failed to process block: %v", err)
		}
		if n := recorder.total(); n != 0 {
			t.Errorf("live tracer fired %d hook(s) with EnableTracer=false: %v", n, recorder.fired())
		}
	})

	t.Run("opted in", func(t *testing.T) {
		// Canonical import: the tracer must still see the whole block.
		chain, recorder := newChain(t)
		_, err := chain.ProcessBlock(context.Background(), parentRoot, blocks[0], ExecuteConfig{
			WriteState:   true,
			WriteHead:    true,
			EnableTracer: true,
		})
		if err != nil {
			t.Fatalf("failed to process block: %v", err)
		}
		for _, hook := range []string{"OnBlockStart", "OnBlockEnd", "OnTxStart", "OnTxEnd", "OnEnter", "OnExit", "OnBalanceChange"} {
			if recorder.fired()[hook] == 0 {
				t.Errorf("%s did not fire with EnableTracer=true; fired: %v", hook, recorder.fired())
			}
		}
	})
}
