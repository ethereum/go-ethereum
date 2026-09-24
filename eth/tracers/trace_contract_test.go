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
	"github.com/ethereum/go-ethereum/internal/ethapi/override"
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

func TestTraceNamespaceCallBlockHashAndOverrides(t *testing.T) {
	// Return NUMBER, BASEFEE and GASPRICE.
	code := common.FromHex("43600052486020523a60405260606000f3")
	api, backend := traceTestAPI(t, code, nil)
	client := traceContractClient(t, api)
	genesis := backend.chain.Genesis().Hash()
	args := map[string]any{"from": traceTestSender, "to": traceTestTarget}
	words := func(values ...int64) []byte {
		var out []byte
		for _, v := range values {
			out = append(out, common.LeftPadBytes(big.NewInt(v).Bytes(), 32)...)
		}
		return out
	}
	// EIP-1898 block selectors, for trace_call and trace_callMany.
	for _, selector := range []any{genesis, map[string]any{"blockHash": genesis}, map[string]any{"blockHash": genesis, "requireCanonical": true}} {
		var result TraceExecution
		if err := client.Call(&result, "trace_call", args, TraceTypes{}, selector); err != nil || !bytes.Equal(result.Output, words(0, 0, 0)) {
			t.Fatalf("trace_call %v: %x %v", selector, []byte(result.Output), err)
		}
		var many []TraceExecution
		if err := client.Call(&many, "trace_callMany", []any{[]any{args, TraceTypes{}}}, selector); err != nil || len(many) != 1 {
			t.Fatalf("trace_callMany %v: %v %v", selector, many, err)
		}
	}
	var result json.RawMessage
	requireTraceCode(t, client.Call(&result, "trace_call", args, TraceTypes{}, common.Hash{1}), -32001)
	requireTraceCode(t, client.Call(&result, "trace_callMany", []any{}, map[string]any{"blockHash": common.Hash{1}}), -32001)

	// Null overrides are accepted.
	var plain TraceExecution
	if err := client.Call(&plain, "trace_call", args, TraceTypes{}, "latest", nil, nil); err != nil || !bytes.Equal(plain.Output, words(0, 0, 0)) {
		t.Fatalf("null overrides: %x %v", []byte(plain.Output), err)
	}
	// State overrides apply before execution, and stateDiff is relative to them.
	unfunded := common.HexToAddress("0xcafe0009")
	state := map[common.Address]any{
		traceTestTarget: map[string]any{"code": hexutil.Bytes(common.FromHex("602a60005260206000f3")), "stateDiff": map[common.Hash]common.Hash{{}: {31: 1}}},
		unfunded:        map[string]any{"balance": "0x1"},
	}
	var overridden TraceExecution
	transfer := map[string]any{"from": unfunded, "to": traceTestTarget, "value": "0x1"}
	if err := client.Call(&overridden, "trace_call", transfer, TraceTypes{"stateDiff"}, "latest", state); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(overridden.Output, words(42)) {
		t.Fatalf("code override: %x", []byte(overridden.Output))
	}
	if diff := overridden.StateDiff[unfunded]; diff == nil || diff.Balance.(map[string]any)["*"].(map[string]any)["from"] != "0x1" {
		t.Fatalf("stateDiff must start from the overridden state: %+v", overridden.StateDiff)
	}
	var many []TraceExecution
	if err := client.Call(&many, "trace_callMany", []any{[]any{transfer, TraceTypes{}}, []any{args, TraceTypes{}}}, "latest", state); err != nil || len(many) != 2 || !bytes.Equal(many[1].Output, words(42)) {
		t.Fatalf("callMany state override: %+v %v", many, err)
	}
	// Block overrides replace the environment, and the zero-fee rule uses the
	// overridden base fee.
	block := map[string]any{"number": "0x10", "baseFeePerGas": "0x7"}
	for fees, want := range map[string][]byte{
		`{}`:                      words(16, 0, 0),
		`{"maxFeePerGas":"0x64"}`: words(16, 7, 7),
		`{"gasPrice":"0x9"}`:      words(16, 7, 9),
		`{"maxFeePerGas":"0x6"}`:  nil,
		`{"gasPrice":"0x0"}`:      words(16, 0, 0),
		`{"maxPriorityFeePerGas":"0x1","maxFeePerGas":"0x64"}`: words(16, 7, 8),
	} {
		call := map[string]any{"from": traceTestSender, "to": traceTestTarget}
		if err := json.Unmarshal([]byte(fees), &call); err != nil {
			t.Fatal(err)
		}
		var result TraceExecution
		err := client.Call(&result, "trace_call", call, TraceTypes{}, "latest", nil, block)
		if want == nil {
			requireTraceCode(t, err, -38012)
			continue
		}
		if err != nil || !bytes.Equal(result.Output, want) {
			t.Fatalf("block override %s: %x %v", fees, []byte(result.Output), err)
		}
		var many []TraceExecution
		if err := client.Call(&many, "trace_callMany", []any{[]any{call, TraceTypes{}}}, "latest", nil, block); err != nil || !bytes.Equal(many[0].Output, want) {
			t.Fatalf("callMany block override %s: %v %v", fees, many, err)
		}
	}
	// Invalid overrides are invalid parameters.
	for _, params := range [][]any{
		{args, TraceTypes{}, "latest", map[common.Address]any{traceTestTarget: map[string]any{"state": map[string]any{}, "stateDiff": map[string]any{}}}},
		{args, TraceTypes{}, "latest", nil, map[string]any{"beaconRoot": common.Hash{}}},
		{args, TraceTypes{}, "latest", nil, nil, nil},
	} {
		requireTraceCode(t, client.Call(&result, "trace_call", params...), -32602)
	}
}

func TestTraceNamespaceCallManyItemErrors(t *testing.T) {
	api, _ := traceTestAPI(t, common.FromHex("00"), nil)
	client := traceContractClient(t, api)
	valid := []any{map[string]any{"from": traceTestSender, "to": traceTestTarget}, TraceTypes{"trace"}}
	for _, tc := range []struct {
		invalid map[string]any
		index   int
		code    int
	}{
		{map[string]any{"from": traceTestSender, "to": traceTestTarget, "gasPrice": "0x1"}, 1, -38012},
		{map[string]any{"from": traceTestSender, "to": traceTestTarget, "gas": "0x5207"}, 2, -38013},
		{map[string]any{"to": traceTestTarget, "data": "0x01", "input": "0x02"}, 0, -32602},
	} {
		calls := []any{valid, valid, valid}
		calls[tc.index] = []any{tc.invalid, TraceTypes{"trace"}}
		var result json.RawMessage
		err := client.Call(&result, "trace_callMany", calls)
		requireTraceCode(t, err, tc.code)
		// One error for the request, with the zero-based item index and no
		// partial results.
		var dataErr rpc.DataError
		if !errors.As(err, &dataErr) {
			t.Fatalf("item %d: error without data: %v", tc.index, err)
		}
		if data, ok := dataErr.ErrorData().(map[string]any); !ok || data["index"] != float64(tc.index) {
			t.Fatalf("item %d: error data %v", tc.index, dataErr.ErrorData())
		}
		if result != nil {
			t.Fatalf("partial result %s", result)
		}
	}
	// Server caps are explicit client-limit errors, never truncation.
	calls := make([]any, traceCallManyLimit+1)
	for i := range calls {
		calls[i] = valid
	}
	var result json.RawMessage
	requireTraceCode(t, client.Call(&result, "trace_callMany", calls), -38026)
}

func TestTraceNamespaceCallManyTransactionBoundaries(t *testing.T) {
	transient := common.HexToAddress("0xcafe0010") // return TLOAD(0), then TSTORE(0, 1)
	counter := common.HexToAddress("0xcafe0011")   // SSTORE(0, SLOAD(0) + 1)
	env := common.HexToAddress("0xcafe0012")       // return NUMBER, TIMESTAMP
	api, _ := traceTestAPI(t, nil, types.GenesisAlloc{
		transient: {Code: common.FromHex("60005c600052600160005d60206000f3")},
		counter:   {Code: common.FromHex("60005460010160005500")},
		env:       {Code: common.FromHex("436000524260205260406000f3")},
	})
	call := func(to *common.Address, data []byte, kinds ...string) TraceCallManyEntry {
		return TraceCallManyEntry{traceTestArgs(to, data), TraceTypes(kinds)}
	}
	// Initcode deploying CALLER SELFDESTRUCT.
	created := crypto.CreateAddress(traceTestSender, 0)
	results, err := api.CallMany(context.Background(), []TraceCallManyEntry{
		call(nil, common.FromHex("6133ff6000526002601ef3"), "trace"),
		call(&transient, nil), call(&transient, nil),
		call(&counter, nil, "trace"), call(&counter, nil, "trace"),
		call(&env, nil), call(&env, nil),
		call(&created, nil, "trace", "stateDiff"),
	}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if have := results[0].Trace[0].Result.(traceCreateResult).Address; have != created {
		t.Fatalf("created %x, want %x", have, created)
	}
	// Transient storage starts empty for every item.
	for _, i := range []int{1, 2} {
		if !bytes.Equal(results[i].Output, make([]byte, 32)) {
			t.Fatalf("item %d: transient storage leaked: %x", i, []byte(results[i].Output))
		}
	}
	// The second increment sees original value 1 and a cold slot, exactly like
	// a separate call on the first increment's post-state.
	slot := map[common.Hash]common.Hash{{}: {31: 1}}
	single, err := api.Call(context.Background(), traceTestArgs(&counter, nil), TraceTypes{"trace"}, nil, &override.StateOverride{counter: override.OverrideAccount{StateDiff: slot}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if have, want := results[4].Trace[0].Result.(traceCallResult).GasUsed, single.Trace[0].Result.(traceCallResult).GasUsed; have != want {
		t.Fatalf("second increment used %d gas, want %d", have, want)
	}
	// Items share one block environment.
	if !bytes.Equal(results[5].Output, results[6].Output) {
		t.Fatalf("environment advanced: %x %x", []byte(results[5].Output), []byte(results[6].Output))
	}
	// EIP-6780: a contract created by an earlier item is not destroyed.
	if len(results[7].Trace) != 2 || results[7].Trace[1].Type != "suicide" {
		t.Fatalf("selfdestruct trace: %+v", results[7].Trace)
	}
	if diff := results[7].StateDiff[created]; diff != nil && diff.Code != "=" {
		t.Fatalf("contract from an earlier item was deleted: %+v", diff)
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
		{"data": "0x01", "input": "0x02"},
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
