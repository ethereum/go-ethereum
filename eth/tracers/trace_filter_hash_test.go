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

package tracers

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

// Replace the chain after the canonical lookup has returned the requested header.
// The block is then traced against a chain it no longer belongs to, so the request
// must fail rather than return the requested hash with the replacement's ancestry.
type traceHashReorgBackend struct {
	*testBackend
	reorg func()
}

func (b *traceHashReorgBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	header, err := b.testBackend.HeaderByNumber(ctx, number)
	if number == 1 && b.reorg != nil {
		reorg := b.reorg
		b.reorg = nil
		reorg()
	}
	return header, err
}

func TestTraceNamespaceFilterBlockHash(t *testing.T) {
	config := *params.AllEthashProtocolChanges
	key, _ := crypto.HexToECDSA(strings.Repeat("0", 63) + "4")
	sender := crypto.PubkeyToAddress(key.PublicKey)
	// Include executed traces exposing NUMBER and TIMESTAMP, as well as rewards.
	genesis := &core.Genesis{Config: &config, GasLimit: 30_000_000, BaseFee: big.NewInt(params.InitialBaseFee), Alloc: types.GenesisAlloc{
		sender:          {Balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil)},
		traceTestTarget: {Code: common.FromHex("436000524260205260406000f3")},
	}}
	backend := newTestBackend(t, 2, genesis, func(i int, block *core.BlockGen) {
		block.SetCoinbase(sender)
		block.AddTx(types.MustSignNewTx(key, types.LatestSigner(&config), &types.LegacyTx{
			Nonce: uint64(i), Gas: 100000, GasPrice: big.NewInt(2 * params.InitialBaseFee), To: &traceTestTarget,
		}))
	})
	defer backend.teardown()
	wrapper := &traceHashReorgBackend{testBackend: backend}
	client := traceContractClient(t, NewTraceAPI(wrapper))
	hash := backend.chain.GetBlockByNumber(1).Hash()
	var want json.RawMessage
	if err := client.Call(&want, "trace_filter", map[string]any{"fromBlock": "0x1", "toBlock": "0x1"}); err != nil {
		t.Fatal(err)
	}
	for _, filter := range []map[string]any{
		{"blockHash": hash},
		{"blockHash": hash, "fromBlock": nil, "toBlock": nil},
		{"blockHash": nil, "fromBlock": "0x1", "toBlock": "0x1"},
		{"blockHash": hash, "fromAddress": []common.Address{sender}, "toAddress": []common.Address{sender}, "mode": "union"},
	} {
		var got json.RawMessage
		if err := client.Call(&got, "trace_filter", filter); err != nil || !bytes.Equal(got, want) {
			t.Fatalf("filter %v: got %s, want %s, err %v", filter, got, want, err)
		}
	}
	for _, filter := range []map[string]any{
		{"blockHash": hash, "count": 0},
		{"blockHash": hash, "after": 2, "count": 1},
		{"blockHash": hash, "fromAddress": []common.Address{traceTestTarget}},
		{"blockHash": backend.chain.Genesis().Hash()},
	} {
		var got json.RawMessage
		if err := client.Call(&got, "trace_filter", filter); err != nil || string(got) != "[]" {
			t.Fatalf("empty filter %v: %s %v", filter, got, err)
		}
	}
	for _, filter := range []map[string]any{
		{"blockHash": hash, "fromBlock": "0x1"},
		{"blockHash": hash, "toBlock": "0x1", "count": 0},
		{"blockHash": "0x01"},
		{"blockHash": map[string]any{"blockHash": hash, "requireCanonical": true}},
		{"fromBlock": hash},
		{"fromBlock": "0x1", "toBlock": hash},
		{"fromBlock": map[string]any{"blockHash": hash}, "toBlock": "0x1"},
		{"fromBlock": "0x1", "toBlock": map[string]any{"blockHash": hash, "requireCanonical": true}},
		{"fromBlock": map[string]any{"blockNumber": "0x1"}, "toBlock": "0x1"},
	} {
		var got json.RawMessage
		requireTraceCode(t, client.Call(&got, "trace_filter", filter), -32602)
	}
	for _, filter := range []map[string]any{{"blockHash": common.Hash{1}}, {"blockHash": common.Hash{1}, "count": 0}} {
		var got json.RawMessage
		requireTraceCode(t, client.Call(&got, "trace_filter", filter), -32001)
	}
	// Generate a longer replacement branch with a different reward recipient.
	_, replacement, _ := core.GenerateChainWithGenesis(genesis, backend.engine, 3, func(_ int, block *core.BlockGen) {
		block.SetCoinbase(traceTestTarget)
	})
	var unknown json.RawMessage
	requireTraceCode(t, client.Call(&unknown, "trace_filter", map[string]any{"blockHash": replacement[0].Hash(), "count": 0}), -32001)
	wrapper.reorg = func() {
		if _, err := backend.chain.InsertChain(replacement); err != nil {
			t.Fatal(err)
		}
	}
	var got json.RawMessage
	requireTraceCode(t, client.Call(&got, "trace_filter", map[string]any{"blockHash": hash}), -32001)
	if backend.chain.GetBlockByNumber(1).Hash() == hash {
		t.Fatal("replacement branch did not become canonical")
	}
	// Once resolved as noncanonical, A is explicitly rejected, even with count zero.
	for _, filter := range []map[string]any{{"blockHash": hash}, {"blockHash": hash, "count": 0}} {
		requireTraceCode(t, client.Call(&got, "trace_filter", filter), -32001)
	}
}

// Model headers and bodies imported ahead of the executed head during sync.
type traceHashUnexecutedBackend struct{ *testBackend }

func (b traceHashUnexecutedBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	if number == rpc.LatestBlockNumber {
		return b.chain.Genesis().Header(), nil
	}
	if number == rpc.SafeBlockNumber || number == rpc.FinalizedBlockNumber {
		return b.chain.CurrentHeader(), nil
	}
	return b.testBackend.HeaderByNumber(ctx, number)
}

func TestTraceNamespaceFilterUnexecutedBlockHash(t *testing.T) {
	backend := newTestBackend(t, 1, &core.Genesis{Config: params.AllEthashProtocolChanges, GasLimit: 30_000_000}, nil)
	defer backend.teardown()
	client := traceContractClient(t, NewTraceAPI(traceHashUnexecutedBackend{backend}))
	hash := backend.chain.GetBlockByNumber(1).Hash()
	for _, filter := range []map[string]any{{"blockHash": hash}, {"blockHash": hash, "count": 0}} {
		var got json.RawMessage
		requireTraceCode(t, client.Call(&got, "trace_filter", filter), -32001)
	}
}

func TestTraceNamespaceFilterExecutedHead(t *testing.T) {
	backend := newTestBackend(t, 1, &core.Genesis{Config: params.AllEthashProtocolChanges, GasLimit: 30_000_000}, nil)
	t.Cleanup(backend.teardown)
	client := traceContractClient(t, NewTraceAPI(traceHashUnexecutedBackend{backend}))
	// The header chain is at block 1, but execution is still at genesis.
	// Latest bounds must select the executed head, which has no traces.
	for _, filter := range []map[string]any{{}, {"fromBlock": "latest"}, {"toBlock": "latest"}, {"fromBlock": "0x0"}} {
		var got json.RawMessage
		if err := client.Call(&got, "trace_filter", filter); err != nil || string(got) != "[]" {
			t.Errorf("latest filter %v: got %s, err %v", filter, got, err)
		}
	}
	// Explicit bounds above execution are invalid, including with count zero.
	for _, filter := range []map[string]any{
		{"fromBlock": "0x1"}, {"toBlock": "0x1"},
		{"fromBlock": "0x1", "count": 0}, {"toBlock": "0x1", "count": 0},
		{"fromBlock": "safe"}, {"toBlock": "finalized", "count": 0},
	} {
		var got json.RawMessage
		requireTraceCode(t, client.Call(&got, "trace_filter", filter), -32602)
	}
}
