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
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/history"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
)

func traceContractClient(t *testing.T, api *TraceAPI) *rpc.Client {
	t.Helper()
	server := rpc.NewServer()
	t.Cleanup(server.Stop)
	if err := server.RegisterName("trace", api); err != nil {
		t.Fatal(err)
	}
	client := rpc.DialInProc(server)
	t.Cleanup(client.Close)
	return client
}

func requireTraceCode(t *testing.T, err error, code int) {
	t.Helper()
	var rpcErr rpc.Error
	if !errors.As(err, &rpcErr) || rpcErr.ErrorCode() != code {
		t.Fatalf("want RPC code %d, got %v", code, err)
	}
}

func TestTraceNamespaceBlockErrors(t *testing.T) {
	api, _ := traceTestAPI(t, nil, nil)
	client := traceContractClient(t, api)
	for _, method := range []string{"trace_call", "trace_callMany", "trace_block", "trace_replayBlockTransactions"} {
		t.Run(method, func(t *testing.T) {
			var args []any
			switch method {
			case "trace_call":
				args = []any{map[string]any{"to": traceTestTarget}, TraceTypes{}, "0xffff"}
			case "trace_callMany":
				args = []any{TraceCalls{}, "0xffff"}
			case "trace_replayBlockTransactions":
				args = []any{"0xffff", TraceTypes{}}
			default:
				args = []any{"0xffff"}
			}
			var result json.RawMessage
			requireTraceCode(t, client.Call(&result, method, args...), -32001)
		})
	}
}

func TestTraceNamespaceRawRejectionCodes(t *testing.T) {
	key, _ := crypto.HexToECDSA(strings.Repeat("0", 63) + "2")
	sender := crypto.PubkeyToAddress(key.PublicKey)
	api, backend := traceTestAPI(t, common.FromHex("60006000fd"), types.GenesisAlloc{sender: {Balance: big.NewInt(1e18), Nonce: 1}})
	client := traceContractClient(t, api)
	// Decodable transactions failing validation use the eth_sendRawTransaction
	// error groups, with -32003 for other rejections.
	for name, code := range map[string]int{"low nonce": 1, "high nonce": 2, "intrinsic gas": 800, "tip": 804, "fee": 806, "balance": 809, "chain": -32003, "signature": -32003, "gas cap": -38026} {
		t.Run(name, func(t *testing.T) {
			var data types.TxData
			legacy := &types.LegacyTx{Nonce: 1, Gas: 100000, GasPrice: big.NewInt(params.InitialBaseFee), To: &traceTestTarget}
			data = legacy
			signer := types.LatestSigner(backend.ChainConfig())
			switch name {
			case "low nonce":
				legacy.Nonce = 0
			case "high nonce":
				legacy.Nonce = 2
			case "intrinsic gas":
				legacy.Gas = 20000
			case "balance":
				legacy.Value = new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil)
			case "fee":
				legacy.GasPrice = big.NewInt(1)
			case "tip":
				data = &types.DynamicFeeTx{ChainID: backend.ChainConfig().ChainID, Nonce: 1, Gas: 100000, GasFeeCap: big.NewInt(params.InitialBaseFee), GasTipCap: big.NewInt(params.InitialBaseFee + 1), To: &traceTestTarget}
			case "chain":
				signer = types.LatestSignerForChainID(big.NewInt(999999))
			case "gas cap":
				legacy.Gas = backend.RPCGasCap() + 1
			}
			tx := types.MustSignNewTx(key, signer, data)
			if name == "signature" {
				tx = types.NewTx(data)
			}
			encoded, _ := tx.MarshalBinary()
			var result json.RawMessage
			requireTraceCode(t, client.Call(&result, "trace_rawTransaction", hexutil.Bytes(encoded), TraceTypes{"trace"}), code)
		})
	}
	var result json.RawMessage
	requireTraceCode(t, client.Call(&result, "trace_rawTransaction", "0x01", TraceTypes{}), -32602)
	// Valid transactions which fail inside the EVM still return execution results.
	for _, gas := range []uint64{21000, 100000} {
		tx := types.MustSignNewTx(key, types.LatestSigner(backend.ChainConfig()), &types.LegacyTx{Nonce: 1, Gas: gas, GasPrice: big.NewInt(params.InitialBaseFee), To: &traceTestTarget})
		encoded, _ := tx.MarshalBinary()
		var result TraceExecution
		if err := client.Call(&result, "trace_rawTransaction", hexutil.Bytes(encoded), TraceTypes{"trace"}); err != nil {
			t.Fatal(err)
		}
		if len(result.Trace) != 1 || result.Trace[0].Error == "" {
			t.Fatalf("missing EVM failure: %+v", result)
		}
	}
}

func TestTraceNamespaceCallRejectionCodes(t *testing.T) {
	api, backend := traceTestAPI(t, common.FromHex("00"), nil)
	client := traceContractClient(t, api)
	base := big.NewInt(params.InitialBaseFee)
	unfunded := common.HexToAddress("0xcafe0009")
	for _, tc := range []struct {
		fields map[string]any
		code   int
	}{
		{map[string]any{"gasPrice": "0x1"}, -38012},
		{map[string]any{"maxFeePerGas": "0x1"}, -38012},
		{map[string]any{"gas": "0x5207"}, -38013},
		{map[string]any{"from": unfunded, "value": "0x1"}, -38014},
		{map[string]any{"from": unfunded, "gasPrice": hexutil.EncodeBig(base)}, -38014},
		{map[string]any{"to": nil, "input": hexutil.Bytes(make([]byte, 1<<20)), "gas": hexutil.Uint64(backend.RPCGasCap())}, -38025},
		{map[string]any{"gas": hexutil.Uint64(backend.RPCGasCap() + 1)}, -38026},
		{map[string]any{"maxFeePerGas": hexutil.EncodeBig(base), "maxPriorityFeePerGas": hexutil.EncodeBig(new(big.Int).Add(base, common.Big1))}, -32602},
	} {
		args := map[string]any{"from": traceTestSender, "to": traceTestTarget}
		for k, v := range tc.fields {
			args[k] = v
		}
		var result json.RawMessage
		err := client.Call(&result, "trace_call", args, TraceTypes{})
		requireTraceCode(t, err, tc.code)
		// trace_callMany reports the same validation codes.
		err = client.Call(&result, "trace_callMany", []any{[]any{args, TraceTypes{}}})
		requireTraceCode(t, err, tc.code)
	}
}

func TestTraceNamespaceCallFields(t *testing.T) {
	api, backend := traceTestAPI(t, common.FromHex("602a60005260206000f3"), nil)
	client := traceContractClient(t, api)
	var baseline TraceExecution
	args := map[string]any{"from": traceTestSender, "to": traceTestTarget, "chainId": hexutil.EncodeBig(backend.ChainConfig().ChainID)}
	if err := client.Call(&baseline, "trace_call", args, TraceTypes{}); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"futureField", "blobs", "proofs"} {
		for _, value := range []any{nil, "ignored"} {
			args[field] = value
			var result TraceExecution
			if err := client.Call(&result, "trace_call", args, TraceTypes{}); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(result.Output, baseline.Output) {
				t.Fatal("unknown field changed execution")
			}
			delete(args, field)
		}
	}
	for _, field := range []string{"gas", "chainId", "blobVersionedHashes", "authorizationList"} {
		var result json.RawMessage
		requireTraceCode(t, client.Call(&result, "trace_call", map[string]any{field: nil}, TraceTypes{}), -32602)
	}
}

func TestTraceNamespaceBlobCall(t *testing.T) {
	api, _ := traceTestAPI(t, common.FromHex("60004960005260206000f3"), nil)
	client := traceContractClient(t, api)
	hash := common.Hash{0: 1, 31: 42}
	for _, kind := range []string{"", "0x03"} {
		args := map[string]any{"from": traceTestSender, "to": traceTestTarget, "blobVersionedHashes": []common.Hash{hash}, "maxFeePerBlobGas": "0x0"}
		if kind != "" {
			args["type"] = kind
		}
		var result TraceExecution
		if err := client.Call(&result, "trace_call", args, TraceTypes{"trace"}); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(result.Output, hash[:]) {
			t.Fatalf("BLOBHASH output: %x", result.Output)
		}
	}
}

func TestTraceNamespaceAuthorizationCall(t *testing.T) {
	key, _ := crypto.HexToECDSA(strings.Repeat("0", 63) + "3")
	authority := crypto.PubkeyToAddress(key.PublicKey)
	api, backend := traceTestAPI(t, common.FromHex("602a60005260206000f3"), types.GenesisAlloc{authority: {Balance: big.NewInt(1)}})
	auth, err := types.SignSetCode(key, types.SetCodeAuthorization{ChainID: *uint256.MustFromBig(backend.ChainConfig().ChainID), Address: traceTestTarget})
	if err != nil {
		t.Fatal(err)
	}
	client := traceContractClient(t, api)
	for _, kind := range []string{"", "0x04"} {
		args := map[string]any{"from": traceTestSender, "to": authority, "authorizationList": []types.SetCodeAuthorization{auth}}
		if kind != "" {
			args["type"] = kind
		}
		var result TraceExecution
		if err := client.Call(&result, "trace_call", args, TraceTypes{"trace", "stateDiff"}); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(result.Output, common.LeftPadBytes([]byte{42}, 32)) || result.StateDiff[authority] == nil {
			t.Fatalf("authorization did not execute: %+v", result)
		}
	}
}

func TestTraceNamespaceConflictingCallFields(t *testing.T) {
	api, _ := traceTestAPI(t, nil, nil)
	client := traceContractClient(t, api)
	for _, fields := range []map[string]any{
		{"type": "0x03"},
		{"type": "0x04"},
		{"type": "0x02", "blobVersionedHashes": []common.Hash{}},
		{"type": "0x03", "authorizationList": []types.SetCodeAuthorization{}},
		{"gasPrice": "0x0", "blobVersionedHashes": []common.Hash{}},
		{"gasPrice": "0x0", "authorizationList": []types.SetCodeAuthorization{}},
		{"blobVersionedHashes": []common.Hash{}, "authorizationList": []types.SetCodeAuthorization{}},
		{"chainId": "0xffff"},
	} {
		fields["to"] = traceTestTarget
		var result json.RawMessage
		requireTraceCode(t, client.Call(&result, "trace_call", fields, TraceTypes{}), -32602)
	}
}

func TestTraceNamespaceFilterBounds(t *testing.T) {
	config := *params.AllEthashProtocolChanges
	backend := newTestBackend(t, 3, &core.Genesis{Config: &config, GasLimit: 30_000_000, BaseFee: big.NewInt(params.InitialBaseFee)}, nil)
	t.Cleanup(backend.teardown)
	client := traceContractClient(t, NewTraceAPI(backend))
	// Bounds beyond the head, reversed ranges (including an explicit toBlock
	// before the omitted latest start) and pending are invalid parameters.
	for _, filter := range []map[string]any{
		{"fromBlock": "0x4"}, {"toBlock": "0x4"}, {"fromBlock": "0x1", "toBlock": "0xffff"},
		{"fromBlock": "0x2", "toBlock": "0x1"}, {"toBlock": "0x2"}, {"fromBlock": "latest", "toBlock": "0x1"},
		{"fromBlock": "pending"}, {"toBlock": "pending"},
	} {
		var result json.RawMessage
		requireTraceCode(t, client.Call(&result, "trace_filter", filter), -32602)
	}
	// Omitted bounds select only the head block.
	for _, filter := range []map[string]any{{}, {"fromBlock": "latest"}, {"toBlock": "latest"}, {"fromBlock": "0x3"}} {
		var frames []map[string]any
		if err := client.Call(&frames, "trace_filter", filter); err != nil {
			t.Fatalf("%v: %v", filter, err)
		}
		if len(frames) != 1 || frames[0]["blockNumber"] != 3.0 || frames[0]["type"] != "reward" {
			t.Fatalf("%v: %+v", filter, frames)
		}
	}
	var frames []map[string]any
	if err := client.Call(&frames, "trace_filter", map[string]any{"fromBlock": "0x1"}); err != nil || len(frames) != 3 {
		t.Fatalf("explicit history search: %+v %v", frames, err)
	}
	// Single-block selectors reject pending too; genesis has no records.
	var result json.RawMessage
	requireTraceCode(t, client.Call(&result, "trace_block", "pending"), -32602)
	requireTraceCode(t, client.Call(&result, "trace_replayBlockTransactions", "pending", TraceTypes{}), -32602)
	for _, method := range []string{"trace_block", "trace_replayBlockTransactions"} {
		args := []any{"0x0"}
		if method != "trace_block" {
			args = append(args, TraceTypes{"trace"})
		}
		if err := client.Call(&result, method, args...); err != nil || string(result) != "[]" {
			t.Fatalf("%s genesis: %s %v", method, result, err)
		}
	}
}

type traceBlockErrorBackend struct {
	Backend
	err error
}

func (b traceBlockErrorBackend) HeaderByNumber(context.Context, rpc.BlockNumber) (*types.Header, error) {
	return nil, b.err
}

func (b traceBlockErrorBackend) BlockByNumber(context.Context, rpc.BlockNumber) (*types.Block, error) {
	return nil, b.err
}

func TestTraceNamespaceBackendErrors(t *testing.T) {
	_, backend := traceTestAPI(t, nil, nil)
	for _, number := range []rpc.BlockNumber{rpc.SafeBlockNumber, rpc.FinalizedBlockNumber} {
		for _, original := range []error{errors.New(number.String() + " block not found"), errors.New("database unavailable"), &traceRPCError{4444, "pruned history"}} {
			api := NewTraceAPI(traceBlockErrorBackend{backend, original})
			_, blockErr := api.Block(context.Background(), number)
			_, filterErr := api.Filter(context.Background(), TraceFilter{FromBlock: &number})
			for i, err := range []error{blockErr, filterErr} {
				if original.Error() == number.String()+" block not found" {
					// trace_filter has no resource-not-found code: an
					// unavailable bound is an invalid range.
					requireTraceCode(t, err, []int{-32001, -32602}[i])
				} else if err != original {
					t.Fatalf("backend error was reclassified: %v -> %v", original, err)
				}
			}
		}
	}
}

func TestTraceNamespacePrunedTransactionLookup(t *testing.T) {
	api, backend := traceTestAPI(t, nil, nil)
	client := traceContractClient(t, api)
	for _, tail := range []uint64{0, 1, 2} {
		rawdb.WriteTxIndexTail(backend.ChainDb(), tail)
		for _, method := range []string{"trace_transaction", "trace_get", "trace_replayTransaction"} {
			args := []any{common.Hash{31: 1}}
			if method != "trace_transaction" {
				args = append(args, []string{})
			}
			var result json.RawMessage
			err := client.Call(&result, method, args...)
			if tail > 1 {
				requireTraceCode(t, err, 4444)
			} else if err != nil || string(result) != "null" {
				t.Fatalf("complete lookup: %s %v", result, err)
			}
		}
	}
}

// Model a node whose history before block 1 is pruned: earliest resolves to
// the cutoff and bodies below it are unavailable.
type traceRetentionBackend struct{ *testBackend }

func (b traceRetentionBackend) HeaderByNumber(ctx context.Context, n rpc.BlockNumber) (*types.Header, error) {
	if n == rpc.EarliestBlockNumber {
		n = 1
	}
	return b.testBackend.HeaderByNumber(ctx, n)
}

func (b traceRetentionBackend) BlockByNumber(ctx context.Context, n rpc.BlockNumber) (*types.Block, error) {
	if n == rpc.EarliestBlockNumber {
		n = 1
	}
	if n == 0 {
		return nil, &history.PrunedHistoryError{}
	}
	return b.testBackend.BlockByNumber(ctx, n)
}

func (b traceRetentionBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	if hash == b.chain.Genesis().Hash() {
		return nil, &history.PrunedHistoryError{}
	}
	return b.testBackend.BlockByHash(ctx, hash)
}

func TestTraceNamespaceEarliestMeansCutoff(t *testing.T) {
	// Block 1 carries only its PoW reward, so tracing it needs no pruned parent body.
	config := *params.AllEthashProtocolChanges
	backend := newTestBackend(t, 2, &core.Genesis{Config: &config, GasLimit: 30_000_000, BaseFee: big.NewInt(params.InitialBaseFee)}, nil)
	t.Cleanup(backend.teardown)
	api := NewTraceAPI(traceRetentionBackend{backend})
	ctx := context.Background()
	earliest, genesisNumber := rpc.EarliestBlockNumber, rpc.BlockNumber(0)
	onlyCutoff := func(frames []*TraceFrame) bool {
		for _, frame := range frames {
			if frame.BlockNumber != 1 {
				return false
			}
		}
		return len(frames) > 0
	}
	frames, err := api.Block(ctx, earliest)
	if err != nil || !onlyCutoff(frames) {
		t.Fatalf("trace_block earliest: %+v %v", frames, err)
	}
	frames, err = api.Filter(ctx, TraceFilter{FromBlock: &earliest, ToBlock: &earliest})
	if err != nil || !onlyCutoff(frames) {
		t.Fatalf("trace_filter earliest: %+v %v", frames, err)
	}
	// Explicit numbers below the cutoff remain unavailable history.
	_, err = api.Block(ctx, genesisNumber)
	requireTraceCode(t, err, 4444)
	_, err = api.Filter(ctx, TraceFilter{FromBlock: &genesisNumber, ToBlock: &earliest})
	requireTraceCode(t, err, 4444)
}
