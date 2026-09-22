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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
)

var traceTestSender = common.HexToAddress("0xcafe0001")
var traceTestTarget = common.HexToAddress("0xcafe0002")

func traceTestAPI(t *testing.T, code []byte, extra types.GenesisAlloc) (*TraceAPI, *testBackend) {
	t.Helper()
	alloc := types.GenesisAlloc{
		traceTestSender: {Balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil)},
		traceTestTarget: {Code: code, Balance: big.NewInt(1)},
	}
	for a, v := range extra {
		alloc[a] = v
	}
	config := *params.AllDevChainProtocolChanges
	backend := newTestBackend(t, 0, &core.Genesis{Config: &config, Alloc: alloc, GasLimit: 30_000_000, Difficulty: big.NewInt(0), BaseFee: big.NewInt(params.InitialBaseFee)}, nil)
	t.Cleanup(backend.teardown)
	return NewTraceAPI(backend), backend
}

func traceTestArgs(to *common.Address, data []byte) TraceCallArgs {
	gas := hexutil.Uint64(1_000_000)
	input := hexutil.Bytes(data)
	return TraceCallArgs{TransactionArgs: ethapi.TransactionArgs{From: &traceTestSender, To: to, Gas: &gas, Input: &input}}
}

func TestTraceNamespaceVMEffects(t *testing.T) {
	// Store 42, MCOPY it to offset 32, return the copied word.
	code := common.FromHex("602a6000526020600060205e60206020f3")
	api, _ := traceTestAPI(t, code, nil)
	result, err := api.Call(context.Background(), traceTestArgs(&traceTestTarget, nil), TraceTypes{"trace", "vmTrace", "stateDiff"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result.Output, common.LeftPadBytes([]byte{42}, 32)) {
		t.Fatalf("output %x", result.Output)
	}
	if result.VMTrace == nil || !bytes.Equal(result.VMTrace.Code, code) {
		t.Fatal("missing executing bytecode")
	}
	found := false
	for _, op := range result.VMTrace.Ops {
		if op.Op == "MCOPY" {
			found = true
			if op.Ex == nil || op.Ex.Mem == nil || op.Ex.Mem.Off != 32 || !bytes.Equal(op.Ex.Mem.Data, result.Output) {
				t.Fatalf("MCOPY effects: %+v", op.Ex)
			}
		}
		if op.Op == "PUSH1" && (op.Ex == nil || len(op.Ex.Push) != 1) {
			t.Fatalf("PUSH effects: %+v", op.Ex)
		}
	}
	if !found {
		t.Fatal("MCOPY was not traced")
	}
}

func TestTraceNamespaceBaseFeeAndEmptySelection(t *testing.T) {
	api, _ := traceTestAPI(t, common.FromHex("4860005260206000f3"), nil)
	for _, kinds := range []TraceTypes{{}, {"stateDiff"}, {"vmTrace"}} {
		result, err := api.Call(context.Background(), traceTestArgs(&traceTestTarget, nil), kinds, nil)
		if err != nil {
			t.Fatal(err)
		}
		if new(big.Int).SetBytes(result.Output).Uint64() != params.InitialBaseFee {
			t.Fatalf("BASEFEE rewritten: %x", result.Output)
		}
		if result.Trace == nil || len(result.Trace) != 0 {
			t.Fatal("unrequested trace must be []")
		}
	}
}

func TestTraceNamespaceSequentialRollback(t *testing.T) {
	// Calldata 0: read; 1: store 42; 2: store 43 then revert.
	code := common.FromHex("60003580600014601e5780602901600055600214601857005b60006000fd5b5060005460005260206000f3")
	api, _ := traceTestAPI(t, code, nil)
	calls := []TraceCallManyEntry{}
	for _, n := range []byte{1, 2, 0} {
		calls = append(calls, TraceCallManyEntry{traceTestArgs(&traceTestTarget, common.LeftPadBytes([]byte{n}, 32)), TraceTypes{"trace", "stateDiff"}})
	}
	result, err := api.CallMany(context.Background(), calls, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 || new(big.Int).SetBytes(result[2].Output).Uint64() != 42 {
		t.Fatalf("sequence: %+v", result)
	}
	if result[1].Trace[0].Error != "Reverted" {
		t.Fatalf("missing revert: %+v", result[1].Trace[0])
	}
	if diff := result[1].StateDiff[traceTestTarget]; diff != nil && len(diff.Storage) != 0 {
		t.Fatalf("reverted storage survived: %+v", diff)
	}
	independent, err := api.Call(context.Background(), traceTestArgs(&traceTestTarget, nil), TraceTypes{}, nil)
	if err != nil || new(big.Int).SetBytes(independent.Output).Sign() != 0 {
		t.Fatalf("simulation leaked: %+v %v", independent, err)
	}
}

func TestTraceNamespaceNestedPrecompile(t *testing.T) {
	for _, value := range []byte{0, 1} {
		// Identity precompile returns 42 into memory at offset 32.
		code := common.FromHex("602a60005360016020600160006000600461fffff15060016020f3")
		code[14] = value
		api, _ := traceTestAPI(t, code, nil)
		result, err := api.Call(context.Background(), traceTestArgs(&traceTestTarget, nil), TraceTypes{"trace", "vmTrace"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(result.Output, []byte{42}) || len(result.Trace) != 1+int(value) || result.Trace[0].Subtraces != uint64(value) {
			t.Fatalf("precompile value %d result: %+v", value, result)
		}
		if value != 0 {
			child := result.Trace[1]
			if len(child.TraceAddress) != 1 || child.TraceAddress[0] != 0 || !bytes.Equal(child.Result.(traceCallResult).Output, []byte{42}) {
				t.Fatalf("child: %+v", child)
			}
		}
		for _, op := range result.VMTrace.Ops {
			if op.Op == "CALL" && (op.Ex == nil || len(op.Ex.Push) != 1 || op.Ex.Push[0].ToInt().Uint64() != 1 || op.Ex.Mem == nil || op.Ex.Mem.Off != 32 || !bytes.Equal(op.Ex.Mem.Data, []byte{42})) {
				t.Fatalf("CALL lost return effects: %+v", op)
			}
		}
	}
	api, _ := traceTestAPI(t, nil, nil)
	identity := common.HexToAddress("0x4")
	root, err := api.Call(context.Background(), traceTestArgs(&identity, []byte{42}), TraceTypes{"trace"}, nil)
	if err != nil || len(root.Trace) != 1 || !bytes.Equal(root.Output, []byte{42}) {
		t.Fatalf("root precompile omitted: %v %v", root, err)
	}
}

func TestTraceNamespaceConstructorAndDelegation(t *testing.T) {
	key, err := crypto.HexToECDSA(strings.Repeat("0", 63) + "1")
	if err != nil {
		t.Fatal(err)
	}
	authority := crypto.PubkeyToAddress(key.PublicKey)
	code := common.FromHex("602a60005260206000f3")
	api, _ := traceTestAPI(t, code, types.GenesisAlloc{authority: {Balance: big.NewInt(1), Code: types.AddressToDelegation(traceTestTarget)}})
	delegated, err := api.Call(context.Background(), traceTestArgs(&authority, nil), TraceTypes{"vmTrace"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if delegated.VMTrace == nil || !bytes.Equal(delegated.VMTrace.Code, code) {
		t.Fatal("delegation marker substituted for executing code")
	}
	init := common.FromHex("600160005360016000f3")
	created, err := api.Call(context.Background(), traceTestArgs(nil, init), TraceTypes{"trace", "vmTrace", "stateDiff"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if created.VMTrace == nil || !bytes.Equal(created.VMTrace.Code, init) {
		t.Fatal("missing initcode")
	}
	frame := created.Trace[0].Result.(traceCreateResult)
	if !bytes.Equal(frame.Code, []byte{1}) {
		t.Fatalf("code: %x", frame.Code)
	}
	diff := created.StateDiff[frame.Address]
	if diff == nil {
		t.Fatal("missing created account")
	}
	if _, ok := diff.Code.(map[string]any)["+"]; !ok {
		t.Fatal("new code must have a creation marker")
	}
}

func TestTraceNamespaceSignedValidationAndAuthorization(t *testing.T) {
	senderKey, _ := crypto.HexToECDSA(strings.Repeat("0", 63) + "2")
	authorityKey, _ := crypto.HexToECDSA(strings.Repeat("0", 63) + "3")
	sender := crypto.PubkeyToAddress(senderKey.PublicKey)
	authority := crypto.PubkeyToAddress(authorityKey.PublicKey)
	api, backend := traceTestAPI(t, common.FromHex("60006000fd"), types.GenesisAlloc{sender: {Balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil)}, authority: {Balance: big.NewInt(1)}})
	chainID := backend.ChainConfig().ChainID
	auth, err := types.SignSetCode(authorityKey, types.SetCodeAuthorization{ChainID: *uint256.MustFromBig(chainID), Address: traceTestTarget})
	if err != nil {
		t.Fatal(err)
	}
	signer := types.LatestSigner(backend.ChainConfig())
	tx, err := types.SignNewTx(senderKey, signer, &types.SetCodeTx{ChainID: uint256.MustFromBig(chainID), GasTipCap: uint256.NewInt(0), GasFeeCap: uint256.NewInt(params.InitialBaseFee), Gas: 200000, To: authority, Value: uint256.NewInt(0), AuthList: []types.SetCodeAuthorization{auth}})
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := tx.MarshalBinary()
	result, err := api.RawTransaction(context.Background(), encoded, TraceTypes{"trace", "stateDiff", "vmTrace"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Trace[0].Error != "Reverted" {
		t.Fatal("delegated call should revert")
	}
	diff := result.StateDiff[authority]
	if diff == nil {
		t.Fatal("authorization was lost on execution revert")
	}
	change, ok := diff.Code.(map[string]any)["*"].(map[string]any)
	if !ok || !bytes.Equal(change["to"].(hexutil.Bytes), types.AddressToDelegation(traceTestTarget)) {
		t.Fatalf("code diff: %+v", diff.Code)
	}
	bad, err := types.SignNewTx(senderKey, signer, &types.LegacyTx{Nonce: 1, Gas: 21000, GasPrice: big.NewInt(params.InitialBaseFee), To: &traceTestTarget})
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ = bad.MarshalBinary()
	if _, err := api.RawTransaction(context.Background(), encoded, TraceTypes{}); err == nil {
		t.Fatal("signed nonce mismatch accepted")
	}
}

func TestTraceNamespaceRPCValidation(t *testing.T) {
	api, _ := traceTestAPI(t, common.FromHex("00"), nil)
	server := rpc.NewServer()
	defer server.Stop()
	if err := server.RegisterName("trace", api); err != nil {
		t.Fatal(err)
	}
	client := rpc.DialInProc(server)
	defer client.Close()
	for _, params := range []string{
		`[null,[]]`, `[{},null]`, `[{},["unknown"]]`, `[{},["trace","trace"]]`, `[{"bogus":1},[]]`,
	} {
		var args []json.RawMessage
		if err := json.Unmarshal([]byte(params), &args); err != nil {
			t.Fatal(err)
		}
		var result json.RawMessage
		err := client.Call(&result, "trace_call", args[0], args[1])
		rpcErr, ok := err.(rpc.Error)
		if !ok || rpcErr.ErrorCode() != -32602 {
			t.Fatalf("%s: %v", params, err)
		}
	}
	for _, params := range []string{`{"mode":"garbage"}`, `{"mode":null}`, `{"mode":1}`, `{"count":-1}`, `{"after":null}`} {
		var result json.RawMessage
		err := client.Call(&result, "trace_filter", json.RawMessage(params))
		if e, ok := err.(rpc.Error); !ok || e.ErrorCode() != -32602 {
			t.Fatalf("filter %s: %v", params, err)
		}
	}
	for _, params := range []string{`{"fromAddress":null,"toAddress":null}`, `{"mode":"intersection"}`, `{"mode":"union"}`} {
		var filter TraceFilter
		if err := json.Unmarshal([]byte(params), &filter); err != nil {
			t.Fatalf("filter %s: %v", params, err)
		}
	}
	// Lookups must retain a literal JSON null, never an empty collection.
	var result json.RawMessage
	if err := client.Call(&result, "trace_transaction", common.Hash{}); err != nil {
		t.Fatal(err)
	}
	if string(result) != "null" {
		t.Fatalf("unknown tx: %s", result)
	}
}

func TestTraceNamespaceCancellation(t *testing.T) {
	api, _ := traceTestAPI(t, common.FromHex("5b600056"), nil)
	for i := 0; i < 20; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		args := traceTestArgs(&traceTestTarget, nil)
		gas := hexutil.Uint64(50_000_000)
		args.Gas = &gas
		_, err := api.Call(ctx, args, TraceTypes{"vmTrace"}, nil)
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("cancelled execution: %v", err)
		}
		// A subsequent request can reuse the EVM pool without inheriting cancellation.
		empty := common.HexToAddress("0xcafe0003")
		if _, err := api.Call(context.Background(), traceTestArgs(&empty, nil), TraceTypes{"trace"}, nil); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTraceNamespaceExplicitCallType(t *testing.T) {
	api, _ := traceTestAPI(t, []byte{0x00}, nil)
	for _, kind := range []hexutil.Uint64{0, 1, 2} {
		args := traceTestArgs(&traceTestTarget, nil)
		args.Type = &kind
		if _, err := api.Call(context.Background(), args, TraceTypes{"trace"}, nil); err != nil {
			t.Fatalf("type %d: %v", kind, err)
		}
	}
	for _, input := range []string{`{"type":"0x00"}`, `{"type":"0x01"}`, `{"type":"0x02"}`} {
		var args TraceCallArgs
		if err := json.Unmarshal([]byte(input), &args); err != nil {
			t.Fatalf("valid byte encoding %s: %v", input, err)
		}
		if args.Type == nil || *args.Type > 2 {
			t.Fatalf("invalid decoded type: %v", args.Type)
		}
	}
	for _, input := range []string{`{"type":"garbage"}`, `{"type":"0x100"}`, `{"type":"0x3"}`, `{"type":true}`} {
		var args TraceCallArgs
		if err := json.Unmarshal([]byte(input), &args); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
	for _, input := range []string{`{"type":"0x0","accessList":[]}`, `{"type":"0x1","maxFeePerGas":"0x0"}`, `{"type":"0x2","gasPrice":"0x0"}`} {
		var args TraceCallArgs
		if err := json.Unmarshal([]byte(input), &args); err != nil {
			t.Fatal(err)
		}
		if _, err := api.Call(context.Background(), args, TraceTypes{}, nil); err == nil {
			t.Fatalf("accepted conflicting fields %s", input)
		}
	}
}
