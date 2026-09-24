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
	"math"
	"math/big"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/internal/ethapi/override"
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
	result, err := api.Call(context.Background(), traceTestArgs(&traceTestTarget, nil), TraceTypes{"trace", "vmTrace", "stateDiff"}, nil, nil, nil)
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

func TestTraceNamespaceVMOperandRanges(t *testing.T) {
	code := common.FromHex("602a600052" + // MSTORE 42 at [0, 32)
		"60005150" + // MLOAD [0, 32)
		"60206040600160" + "1f60006004" + "61fffff150" + // identity returns 1 byte into the window [64, 96)
		"600060006001601f60006004" + "61fffff150" + // CALL without an output window
		"600060005d" + // TSTORE
		"6000600055" + // SSTORE leaving slot 0 unchanged
		"6001600155") // SSTORE, then run off the end of the code
	api, _ := traceTestAPI(t, code, nil)
	result, err := api.Call(context.Background(), traceTestArgs(&traceTestTarget, nil), TraceTypes{"vmTrace"}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var mems, stores []string
	for _, op := range result.VMTrace.Ops {
		if op.PC >= uint64(len(code)) {
			t.Fatalf("synthetic %s at pc %d", op.Op, op.PC)
		}
		if op.Ex == nil {
			t.Fatalf("%s at pc %d has no effects", op.Op, op.PC)
		}
		switch op.Op {
		case "MLOAD", "CALL":
			mem, _ := json.Marshal(op.Ex.Mem)
			mems = append(mems, string(mem))
		case "TSTORE", "SSTORE":
			store, _ := json.Marshal(op.Ex.Store)
			stores = append(stores, string(store))
		}
	}
	word := hexutil.Encode(common.LeftPadBytes([]byte{42}, 32))
	window := hexutil.Encode(common.RightPadBytes([]byte{42}, 32))
	if want := []string{`{"off":0,"data":"` + word + `"}`, `{"off":64,"data":"` + window + `"}`, `null`}; !slices.Equal(mems, want) {
		t.Fatalf("mem:\nhave %v\nwant %v", mems, want)
	}
	if want := []string{`null`, `{"key":"0x0","val":"0x0"}`, `{"key":"0x1","val":"0x1"}`}; !slices.Equal(stores, want) {
		t.Fatalf("store:\nhave %v\nwant %v", stores, want)
	}
	if last := result.VMTrace.Ops[len(result.VMTrace.Ops)-1]; last.Op != "SSTORE" {
		t.Fatalf("last op %s", last.Op)
	}
}

func TestTraceNamespaceVMFailingOps(t *testing.T) {
	for _, tc := range []struct {
		name, code string
		ops        []string // opcodes in the trace; the last one halted if halted is set
		halted     bool
	}{
		// Operations rejected before execution are omitted.
		{"undefined opcode", "60010c", []string{"PUSH1"}, false},
		{"stack underflow", "600101", []string{"PUSH1"}, false},
		// Operations that began executing keep their cost with ex and sub null.
		{"bad jump", "600556", []string{"PUSH1", "JUMP"}, true},
		{"out of gas", "6001600155", []string{"PUSH1", "PUSH1", "SSTORE"}, true},
	} {
		api, _ := traceTestAPI(t, common.FromHex(tc.code), nil)
		args := traceTestArgs(&traceTestTarget, nil)
		gas := hexutil.Uint64(30_000)
		args.Gas = &gas
		result, err := api.Call(context.Background(), args, TraceTypes{"vmTrace"}, nil, nil, nil)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		var ops []string
		for i, op := range result.VMTrace.Ops {
			ops = append(ops, op.Op)
			last := i == len(result.VMTrace.Ops)-1
			if halted := last && tc.halted; (op.Ex == nil) != halted || op.Sub != nil || op.Cost == 0 && halted {
				t.Errorf("%s: %s ex %+v sub %+v cost %d", tc.name, op.Op, op.Ex, op.Sub, op.Cost)
			}
		}
		if !slices.Equal(ops, tc.ops) {
			t.Fatalf("%s: ops %v, want %v", tc.name, ops, tc.ops)
		}
	}
	// Stack overflow: the rejected push is omitted and every traced op completed.
	api, _ := traceTestAPI(t, common.FromHex("5b5f5f56"), nil)
	result, err := api.Call(context.Background(), traceTestArgs(&traceTestTarget, nil), TraceTypes{"vmTrace"}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range result.VMTrace.Ops {
		if op.Ex == nil {
			t.Fatalf("stack overflow: %s at pc %d has no effects", op.Op, op.PC)
		}
	}
}

func TestTraceNamespaceBaseFeeAndEmptySelection(t *testing.T) {
	// Return BASEFEE, GASPRICE and BLOBBASEFEE.
	api, _ := traceTestAPI(t, common.FromHex("486000523a6020524a60405260606000f3"), nil)
	base := big.NewInt(params.InitialBaseFee)
	quantity := func(n *big.Int) *hexutil.Big { return (*hexutil.Big)(n) }
	blobHashes := []common.Hash{{0: 1}}
	for _, tc := range []struct {
		name                           string
		modify                         func(*TraceCallArgs)
		baseFee, gasPrice, blobBaseFee *big.Int
	}{
		// Omitted and zero fees select a zero effective price, as in eth_call:
		// GASPRICE and BASEFEE are both zero.
		{"omitted", func(*TraceCallArgs) {}, common.Big0, common.Big0, common.Big1},
		{"zero gasPrice", func(a *TraceCallArgs) { a.GasPrice = quantity(common.Big0) }, common.Big0, common.Big0, common.Big1},
		{"zero caps", func(a *TraceCallArgs) {
			a.MaxFeePerGas, a.MaxPriorityFeePerGas = quantity(common.Big0), quantity(common.Big0)
		}, common.Big0, common.Big0, common.Big1},
		// Positive prices keep the block's base fee.
		{"gasPrice", func(a *TraceCallArgs) { a.GasPrice = quantity(base) }, base, base, common.Big1},
		{"fee cap", func(a *TraceCallArgs) { a.MaxFeePerGas = quantity(new(big.Int).Mul(base, common.Big2)) }, base, base, common.Big1},
		// BLOBBASEFEE is zero when maxFeePerBlobGas is defaulted or supplied as zero.
		{"defaulted blob fee", func(a *TraceCallArgs) { a.BlobHashes = blobHashes }, common.Big0, common.Big0, common.Big0},
		{"zero blob fee", func(a *TraceCallArgs) { a.BlobHashes, a.BlobFeeCap = blobHashes, quantity(common.Big0) }, common.Big0, common.Big0, common.Big0},
		{"blob fee", func(a *TraceCallArgs) { a.BlobHashes, a.BlobFeeCap = blobHashes, quantity(common.Big1) }, common.Big0, common.Big0, common.Big1},
	} {
		args := traceTestArgs(&traceTestTarget, nil)
		tc.modify(&args)
		result, err := api.Call(context.Background(), args, TraceTypes{}, nil, nil, nil)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		want := append(append(common.LeftPadBytes(tc.baseFee.Bytes(), 32), common.LeftPadBytes(tc.gasPrice.Bytes(), 32)...), common.LeftPadBytes(tc.blobBaseFee.Bytes(), 32)...)
		if !bytes.Equal(result.Output, want) {
			t.Fatalf("%s: BASEFEE, GASPRICE, BLOBBASEFEE\nhave %x\nwant %x", tc.name, []byte(result.Output), want)
		}
	}
	for _, kinds := range []TraceTypes{{}, {"stateDiff"}, {"vmTrace"}} {
		result, err := api.Call(context.Background(), traceTestArgs(&traceTestTarget, nil), kinds, nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if result.Trace == nil || len(result.Trace) != 0 {
			t.Fatal("unrequested trace must be []")
		}
	}
}

func TestTraceNamespaceIgnoresSuppliedNonce(t *testing.T) {
	api, _ := traceTestAPI(t, nil, nil)
	for _, nonce := range []uint64{0, 5} {
		args := traceTestArgs(nil, common.FromHex("600160005360016000f3"))
		supplied := hexutil.Uint64(nonce)
		args.Nonce = &supplied
		result, err := api.Call(context.Background(), args, TraceTypes{"trace", "stateDiff"}, nil, nil, nil)
		if err != nil {
			t.Fatalf("nonce %d: %v", nonce, err)
		}
		// The sender's state nonce is 0; the supplied nonce is neither
		// validated nor used for the created address.
		want := crypto.CreateAddress(traceTestSender, 0)
		if have := result.Trace[0].Result.(traceCreateResult).Address; have != want {
			t.Fatalf("nonce %d: created %x, want %x", nonce, have, want)
		}
		if change, ok := result.StateDiff[traceTestSender].Nonce.(map[string]any)["*"].(map[string]any); !ok || change["to"] != hexutil.Uint64(1) {
			t.Fatalf("nonce %d: sender nonce diff %+v", nonce, result.StateDiff[traceTestSender].Nonce)
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
	result, err := api.CallMany(context.Background(), calls, nil, nil, nil)
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
	independent, err := api.Call(context.Background(), traceTestArgs(&traceTestTarget, nil), TraceTypes{}, nil, nil, nil)
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
		result, err := api.Call(context.Background(), traceTestArgs(&traceTestTarget, nil), TraceTypes{"trace", "vmTrace"}, nil, nil, nil)
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
	root, err := api.Call(context.Background(), traceTestArgs(&identity, []byte{42}), TraceTypes{"trace"}, nil, nil, nil)
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
	delegated, err := api.Call(context.Background(), traceTestArgs(&authority, nil), TraceTypes{"vmTrace"}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if delegated.VMTrace == nil || !bytes.Equal(delegated.VMTrace.Code, code) {
		t.Fatal("delegation marker substituted for executing code")
	}
	init := common.FromHex("600160005360016000f3")
	created, err := api.Call(context.Background(), traceTestArgs(nil, init), TraceTypes{"trace", "vmTrace", "stateDiff"}, nil, nil, nil)
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
		`[null,[]]`, `[{},null]`, `[{},["unknown"]]`, `[{},["trace","trace"]]`,
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

func TestTraceNamespaceErrorLabels(t *testing.T) {
	inner := common.HexToAddress("0xcafe0020")
	// Call inner with STATICCALL, then stop.
	static := "6000600060006000" + "73" + common.Bytes2Hex(inner.Bytes()) + "5afa5000"
	create2 := "6000600060006000f550"
	for _, tc := range []struct {
		name, code, inner string
		to                *common.Address
		input             []byte
		nonce, gas        uint64
		frame             int
		want              string
	}{
		{name: "revert", code: "60006000fd", want: "Reverted"},
		{name: "out of gas", code: "5b600056", want: "Out of gas"},
		{name: "designated invalid", code: "fe", want: "Bad instruction"},
		{name: "undefined opcode", code: "0c", want: "Bad instruction"},
		{name: "bad jump", code: "600556", want: "Bad jump destination"},
		{name: "stack underflow", code: "01", want: "Stack underflow"},
		{name: "stack overflow", code: "5b5f5f56", want: "Out of stack"},
		{name: "static sstore", code: static, inner: "6001600055", frame: 1, want: "Mutable Call In Static Context"},
		{name: "static tstore", code: static, inner: "600160005d", frame: 1, want: "Mutable Call In Static Context"},
		{name: "returndata bounds", code: "6001600060003e", want: "Out of bounds"},
		{name: "precompile", to: &common.Address{19: 6}, input: bytes.Repeat([]byte{1}, 128), want: "Built-in failed"},
		{name: "collision", code: create2 + create2 + "00", frame: 2, want: "Contract address collision"},
		{name: "code deposit", input: common.FromHex("6110006000f3"), want: "Out of gas"},
		{name: "code size", input: common.FromHex("62010001" + "6000f3"), gas: 25_000_000, want: "Code size limit exceeded"},
		{name: "code prefix", input: common.FromHex("60ef60005360016000f3"), want: "Invalid code prefix 0xEF"},
		{name: "nonce overflow", code: "600060006000f000", nonce: math.MaxUint64, frame: 1, want: "Nonce overflow"},
	} {
		api, _ := traceTestAPI(t, nil, nil)
		overrides := override.StateOverride{}
		to := tc.to
		if tc.code != "" {
			code := hexutil.Bytes(common.FromHex(tc.code))
			account := override.OverrideAccount{Code: &code}
			if tc.nonce != 0 {
				account.Nonce = (*hexutil.Uint64)(&tc.nonce)
			}
			overrides[traceTestTarget] = account
			to = &traceTestTarget
		}
		if tc.inner != "" {
			code := hexutil.Bytes(common.FromHex(tc.inner))
			overrides[inner] = override.OverrideAccount{Code: &code}
		}
		args := traceTestArgs(to, tc.input)
		gas := hexutil.Uint64(100_000)
		if tc.gas != 0 {
			gas = hexutil.Uint64(tc.gas)
		}
		args.Gas = &gas
		result, err := api.Call(context.Background(), args, TraceTypes{"trace"}, nil, &overrides, nil)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if len(result.Trace) <= tc.frame {
			t.Fatalf("%s: missing frame %d", tc.name, tc.frame)
		}
		if have := result.Trace[tc.frame].Error; have != tc.want {
			t.Errorf("%s: frame %d error %q, want %q", tc.name, tc.frame, have, tc.want)
			continue
		}
		frame := result.Trace[tc.frame]
		// REVERT keeps {gasUsed, output}; exceptional halts have no result.
		if _, ok := frame.Result.(traceCallResult); ok != (tc.want == "Reverted") {
			t.Fatalf("%s: result %+v", tc.name, frame.Result)
		}
	}
}

func TestTraceNamespaceCancellation(t *testing.T) {
	api, _ := traceTestAPI(t, common.FromHex("5b600056"), nil)
	for i := 0; i < 20; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		// Omitted gas runs up to the RPC gas cap.
		args := traceTestArgs(&traceTestTarget, nil)
		args.Gas = nil
		_, err := api.Call(ctx, args, TraceTypes{"vmTrace"}, nil, nil, nil)
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("cancelled execution: %v", err)
		}
		// A subsequent request can reuse the EVM pool without inheriting cancellation.
		empty := common.HexToAddress("0xcafe0003")
		if _, err := api.Call(context.Background(), traceTestArgs(&empty, nil), TraceTypes{"trace"}, nil, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTraceNamespaceExplicitCallType(t *testing.T) {
	api, _ := traceTestAPI(t, []byte{0x00}, nil)
	for _, kind := range []hexutil.Uint64{0, 1, 2} {
		args := traceTestArgs(&traceTestTarget, nil)
		args.Type = &kind
		if _, err := api.Call(context.Background(), args, TraceTypes{"trace"}, nil, nil, nil); err != nil {
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
	for _, input := range []string{`{"type":"garbage"}`, `{"type":"0x100"}`, `{"type":"0x5"}`, `{"type":true}`} {
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
		if _, err := api.Call(context.Background(), args, TraceTypes{}, nil, nil, nil); err == nil {
			t.Fatalf("accepted conflicting fields %s", input)
		}
	}
}

func TestTraceNamespaceStorageMarkers(t *testing.T) {
	// Set slot 0 from zero to 42 and clear slot 1 on an account that survives.
	code := common.FromHex("602a6000556000600155")
	slot0, slot1 := common.Hash{}, common.Hash{31: 1}
	api, _ := traceTestAPI(t, code, types.GenesisAlloc{traceTestTarget: {Code: code, Balance: big.NewInt(1), Storage: map[common.Hash]common.Hash{slot1: {31: 7}}}})
	result, err := api.Call(context.Background(), traceTestArgs(&traceTestTarget, nil), TraceTypes{"stateDiff"}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	have, _ := json.Marshal(result.StateDiff[traceTestTarget].Storage)
	want, _ := json.Marshal(map[common.Hash]any{
		slot0: map[string]any{"*": map[string]any{"from": common.Hash{}, "to": common.Hash{31: 42}}},
		slot1: map[string]any{"*": map[string]any{"from": common.Hash{31: 7}, "to": common.Hash{}}},
	})
	if !bytes.Equal(have, want) {
		t.Fatalf("surviving account storage:\nhave %s\nwant %s", have, want)
	}
	// A born account's storage carries the creation marker.
	created, err := api.Call(context.Background(), traceTestArgs(nil, common.FromHex("602a60005500")), TraceTypes{"trace", "stateDiff"}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	address := created.Trace[0].Result.(traceCreateResult).Address
	have, _ = json.Marshal(created.StateDiff[address].Storage)
	want, _ = json.Marshal(map[common.Hash]any{slot0: map[string]any{"+": common.Hash{31: 42}}})
	if !bytes.Equal(have, want) {
		t.Fatalf("born account storage:\nhave %s\nwant %s", have, want)
	}
}

func TestTraceNamespaceEmptyVMFrames(t *testing.T) {
	empty := common.HexToAddress("0xcafe0003")
	call := func(to common.Address, value byte) []byte {
		code := common.FromHex("60006000600060006000")
		code[9] = value
		code = append(append(append(code, 0x73), to.Bytes()...), common.FromHex("61fffff150")...)
		return code
	}
	// Enter an empty-code account and a precompile, then fail a value transfer
	// precondition: the target only holds one wei.
	code := append(append(append(call(empty, 0), call(common.HexToAddress("0x4"), 0)...), call(empty, 2)...), 0x00)
	api, _ := traceTestAPI(t, code, nil)
	for _, to := range []common.Address{empty, common.HexToAddress("0x4")} {
		result, err := api.Call(context.Background(), traceTestArgs(&to, nil), TraceTypes{"vmTrace"}, nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if have, _ := json.Marshal(result.VMTrace); string(have) != `{"code":"0x","ops":[]}` {
			t.Fatalf("root vmTrace for %x: %s", to, have)
		}
	}
	result, err := api.Call(context.Background(), traceTestArgs(&traceTestTarget, nil), TraceTypes{"vmTrace"}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var subs []string
	for _, op := range result.VMTrace.Ops {
		if op.Op == "CALL" {
			sub, _ := json.Marshal(op.Sub)
			subs = append(subs, string(sub))
		}
	}
	if want := []string{`{"code":"0x","ops":[]}`, `{"code":"0x","ops":[]}`, `null`}; !slices.Equal(subs, want) {
		t.Fatalf("CALL subs: have %v, want %v", subs, want)
	}
}

func TestTraceNamespacePrecheckFailuresEmitNoFrame(t *testing.T) {
	empty := common.HexToAddress("0xcafe0003")
	call := func(value string) string {
		return "60006000600060006" + value + "73" + common.Bytes2Hex(empty.Bytes()) + "61fffff150"
	}
	// The target holds one wei: a CALL and a CREATE transferring two wei fail
	// their balance precheck, then a zero-value CALL succeeds.
	code := common.FromHex(call("002") + "600060006002f050" + call("000") + "00")
	api, _ := traceTestAPI(t, code, nil)
	result, err := api.Call(context.Background(), traceTestArgs(&traceTestTarget, nil), TraceTypes{"trace", "vmTrace"}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Trace) != 2 || result.Trace[0].Subtraces != 1 {
		t.Fatalf("precheck failures must not emit frames: %+v", result.Trace)
	}
	child := result.Trace[1]
	if !slices.Equal(child.TraceAddress, []uint64{0}) || child.Error != "" || child.Action.(traceCallAction).To != empty {
		t.Fatalf("child path: %+v", child)
	}
	var subs []bool
	for _, op := range result.VMTrace.Ops {
		if op.Op == "CALL" || op.Op == "CREATE" {
			subs = append(subs, op.Sub != nil)
		}
	}
	if !slices.Equal(subs, []bool{false, false, true}) {
		t.Fatalf("VM subs: %v", subs)
	}
}

func TestTraceNamespaceCreateCost(t *testing.T) {
	// Store initcode PUSH1 42 PUSH1 0 SSTORE STOP at memory offset 26, then run
	// it through CREATE and CREATE2 (salt 0).
	code := common.FromHex("65602a600055006000526006601a6000f05060006006601a6000f55000")
	config := *params.AllDevChainProtocolChanges
	config.OsakaTime = nil
	config.BogotaTime = nil
	backend := newTestBackend(t, 0, &core.Genesis{Config: &config, GasLimit: 30_000_000, Difficulty: big.NewInt(0), BaseFee: big.NewInt(params.InitialBaseFee), Alloc: types.GenesisAlloc{
		traceTestSender: {Balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil)}, traceTestTarget: {Code: code},
	}}, nil)
	t.Cleanup(backend.teardown)
	result, err := NewTraceAPI(backend).Call(context.Background(), traceTestArgs(&traceTestTarget, nil), TraceTypes{"trace", "vmTrace"}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Prague base costs: 32000, one initcode word, and one hashed word for CREATE2.
	base := map[string]uint64{"CREATE": 32002, "CREATE2": 32008}
	creates := 0
	ops := result.VMTrace.Ops
	for i, op := range ops {
		if base[op.Op] == 0 {
			continue
		}
		creates++
		forwarded := uint64(result.Trace[creates].Action.(traceCreateAction).Gas)
		if op.Cost != base[op.Op]+forwarded {
			t.Fatalf("%s cost %d, want %d + %d forwarded", op.Op, op.Cost, base[op.Op], forwarded)
		}
		if op.Sub == nil || len(op.Sub.Ops) == 0 || op.Ex == nil {
			t.Fatalf("%s missing child trace or effects: %+v", op.Op, op)
		}
		leftover := op.Sub.Ops[len(op.Sub.Ops)-1].Ex.Used
		if prev := ops[i-1].Ex.Used; op.Ex.Used != prev-op.Cost+leftover {
			t.Fatalf("%s used %d, want %d - %d + %d", op.Op, op.Ex.Used, prev, op.Cost, leftover)
		}
	}
	if creates != 2 {
		t.Fatalf("traced %d creations", creates)
	}
}
