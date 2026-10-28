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

package traceapi

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

func TestMinedPathsReplayAndFilter(t *testing.T) {
	key, _ := crypto.HexToECDSA(strings.Repeat("0", 63) + "4")
	sender := crypto.PubkeyToAddress(key.PublicKey)
	child := common.HexToAddress("0xcafe0003")
	miner := common.HexToAddress("0xcafe0004")
	childCode := common.FromHex("602a60005260206000f3")
	parentCode := append(common.FromHex("6000600060006000600073"), child.Bytes()...)
	parentCode = append(parentCode, common.FromHex("61fffff15000")...)
	config := *params.AllEthashProtocolChanges
	genesis := &core.Genesis{Config: &config, GasLimit: 30_000_000, BaseFee: big.NewInt(params.InitialBaseFee), Alloc: types.GenesisAlloc{
		sender:     {Balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil)},
		testTarget: {Balance: big.NewInt(0), Code: parentCode}, child: {Balance: big.NewInt(0), Code: childCode},
	}}
	var hashes []common.Hash
	backend := newTestBackend(t, 2, genesis, func(i int, b *core.BlockGen) {
		b.SetCoinbase(miner)
		tx := types.MustSignNewTx(key, types.LatestSigner(&config), &types.LegacyTx{Nonce: uint64(i), Gas: 200000, GasPrice: big.NewInt(2 * params.InitialBaseFee), To: &testTarget})
		b.AddTx(tx)
		hashes = append(hashes, tx.Hash())
	})
	defer backend.teardown()
	api := NewAPI(backend)
	ctx := context.Background()
	root, err := api.Get(ctx, hashes[0], Path{})
	if err != nil {
		t.Fatal(err)
	}
	if root == nil || len(root.TraceAddress) != 0 || root.Subtraces != 1 {
		t.Fatalf("root: %+v", root)
	}
	nested, err := api.Get(ctx, hashes[0], Path{0})
	if err != nil {
		t.Fatal(err)
	}
	if nested == nil || nested.Action.(callAction).To != child {
		t.Fatalf("child: %+v", nested)
	}
	missing, err := api.Get(ctx, hashes[0], Path{0, 0})
	if err != nil || missing != nil {
		t.Fatalf("missing: %+v %v", missing, err)
	}
	replay, err := api.ReplayTransaction(ctx, hashes[0], Outputs{"trace", "vmTrace", "stateDiff"})
	if err != nil {
		t.Fatal(err)
	}
	if replay.TransactionHash == nil || *replay.TransactionHash != hashes[0] {
		t.Fatal("replay identity missing")
	}
	if len(replay.Trace) != 2 || replay.Trace[0].location != nil {
		t.Fatal("replay must contain unlocalized frames")
	}
	childVM := false
	for _, op := range replay.VMTrace.Ops {
		if op.Op == "CALL" {
			childVM = op.Sub != nil && bytes.Equal(op.Sub.Code, childCode)
		}
	}
	if !childVM {
		t.Fatal("nested VM code missing")
	}
	blockReplay, err := api.ReplayBlockTransactions(ctx, 1, Outputs{"trace"})
	if err != nil {
		t.Fatal(err)
	}
	if len(blockReplay) != 1 || *blockReplay[0].TransactionHash != hashes[0] {
		t.Fatalf("block replay: %+v", blockReplay)
	}
	traces, err := api.Block(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(traces) != 3 || traces[2].Type != "reward" || traces[2].TransactionHash != nil || traces[2].TransactionPosition != nil {
		t.Fatalf("reward localization: %+v", traces)
	}
	if traces[2].Action.(rewardAction).Value.ToInt().Cmp(big.NewInt(2e18)) != 0 {
		t.Fatal("wrong PoW issuance")
	}
	from, to := rpc.BlockNumber(1), rpc.BlockNumber(2)
	filter := FilterArgs{FromBlock: &from, ToBlock: &to, FromAddress: []common.Address{sender}, ToAddress: []common.Address{child}}
	frames, err := api.Filter(ctx, filter)
	if err != nil || len(frames) != 0 {
		t.Fatalf("filters must intersect: %+v %v", frames, err)
	}
	intersection := len(frames)
	filter.Mode = FilterUnion
	frames, err = api.Filter(ctx, filter)
	if err != nil || len(frames) <= intersection {
		t.Fatalf("union must widen the intersection: %+v %v", frames, err)
	}
	for _, frame := range frames {
		if !matchesFilter(frame, FilterArgs{FromAddress: filter.FromAddress}) && !matchesFilter(frame, FilterArgs{ToAddress: filter.ToAddress}) {
			t.Fatalf("union returned a frame matching neither list: %+v", frame)
		}
	}
	// A one-sided union selects the same frames as the default.
	oneSided := FilterArgs{FromBlock: &from, ToBlock: &to, FromAddress: filter.FromAddress}
	plain, err := api.Filter(ctx, oneSided)
	if err != nil {
		t.Fatal(err)
	}
	oneSided.Mode = FilterUnion
	if union, err := api.Filter(ctx, oneSided); err != nil || len(union) != len(plain) || len(plain) == 0 {
		t.Fatalf("one-sided union: %d vs %d %v", len(union), len(plain), err)
	}
	filter.Mode = ""
	filter.FromAddress = []common.Address{testTarget}
	after, count := uint64(1), uint64(1)
	filter.After = &after
	filter.Count = &count
	frames, err = api.Filter(ctx, filter)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 1 || frames[0].BlockNumber != 2 || *frames[0].TransactionHash != hashes[1] {
		t.Fatalf("pagination: %+v", frames)
	}
	filter = FilterArgs{FromBlock: &from, ToBlock: &to, ToAddress: []common.Address{miner}}
	frames, err = api.Filter(ctx, filter)
	if err != nil || len(frames) != 2 || frames[0].Type != "reward" {
		t.Fatalf("reward filtering: %+v %v", frames, err)
	}
}

func TestSelfDestructForkRules(t *testing.T) {
	beneficiary := common.HexToAddress("0xcafe0005")
	// Overwrite slot 0, then selfdestruct.
	code := append(common.FromHex("6005600055"), 0x73)
	code = append(append(code, beneficiary.Bytes()...), 0xff)
	for _, postCancun := range []bool{false, true} {
		config := *params.AllDevChainProtocolChanges
		if !postCancun {
			config.CancunTime = nil
			config.PragueTime = nil
			config.OsakaTime = nil
			config.BogotaTime = nil
		}
		backend := newTestBackend(t, 0, &core.Genesis{Config: &config, GasLimit: 30_000_000, Difficulty: big.NewInt(0), BaseFee: big.NewInt(params.InitialBaseFee), Alloc: types.GenesisAlloc{
			testSender: {Balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil)}, testTarget: {Balance: big.NewInt(7), Code: code, Storage: map[common.Hash]common.Hash{{}: {31: 7}}},
		}}, nil)
		api := NewAPI(backend)
		result, err := api.Call(context.Background(), testCallArgs(&testTarget, nil), Outputs{"trace", "stateDiff", "vmTrace"}, nil, nil, nil)
		backend.teardown()
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Trace) != 2 || result.Trace[1].Type != "suicide" {
			t.Fatalf("selfdestruct missing: %+v", result.Trace)
		}
		diff := result.StateDiff[testTarget]
		if diff == nil {
			t.Fatal("missing account transition")
		}
		// The beneficiary did not exist before it was credited.
		created := `{"balance":{"+":"0x7"},"nonce":{"+":"0x0"},"code":{"+":"0x"},"storage":{}}`
		if have, _ := json.Marshal(result.StateDiff[beneficiary]); string(have) != created {
			t.Fatalf("beneficiary: have %s, want %s", have, created)
		}
		if postCancun {
			if diff.Code != "=" {
				t.Fatalf("Cancun deleted code: %+v", diff.Code)
			}
		} else {
			change, ok := diff.Code.(map[string]any)
			if !ok || !bytes.Equal(change["-"].(hexutil.Bytes), code) {
				t.Fatalf("pre-Cancun deletion: %+v", diff.Code)
			}
			// Deletion wipes all storage; slots are not listed.
			if diff.Storage == nil || len(diff.Storage) != 0 {
				t.Fatalf("deleted account storage: %+v", diff.Storage)
			}
		}
	}
}

func TestTouchedEmptyAccountDeletion(t *testing.T) {
	empty := common.HexToAddress("0xcafe0006")
	// CALL the empty account without value.
	code := append(common.FromHex("600060006000600060007"+"3"), empty.Bytes()...)
	code = append(code, common.FromHex("61fffff15000")...)
	config := *params.AllDevChainProtocolChanges
	// Genesis keeps EIP-161-empty accounts, including the zero-address coinbase.
	backend := newTestBackend(t, 0, &core.Genesis{Config: &config, GasLimit: 30_000_000, Difficulty: big.NewInt(0), BaseFee: big.NewInt(params.InitialBaseFee), Alloc: types.GenesisAlloc{
		testSender: {Balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil)}, testTarget: {Code: code},
		empty: {Balance: common.Big0}, common.Address{}: {Balance: common.Big0},
	}}, nil)
	t.Cleanup(backend.teardown)
	api := NewAPI(backend)
	deleted := `{"balance":{"-":"0x0"},"nonce":{"-":"0x0"},"code":{"-":"0x"},"storage":{}}`
	// A zero-fee call pays no fee but touches the coinbase as a block does, and so does paying a zero
	// tip: either way the empty coinbase is deleted.
	priced := testCallArgs(&testTarget, nil)
	priced.GasPrice = (*hexutil.Big)(big.NewInt(params.InitialBaseFee))
	for _, args := range []CallArgs{testCallArgs(&testTarget, nil), priced} {
		result, err := api.Call(context.Background(), args, Outputs{"stateDiff"}, nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, address := range []common.Address{empty, {}} {
			if have, _ := json.Marshal(result.StateDiff[address]); string(have) != deleted {
				t.Fatalf("%x: have %s, want %s", address, have, deleted)
			}
		}
	}
}

func TestBlobFeeAccounting(t *testing.T) {
	api, _ := newTestAPI(t, common.FromHex("00"), nil)
	args := testCallArgs(&testTarget, nil)
	args.MaxFeePerGas = (*hexutil.Big)(big.NewInt(params.InitialBaseFee))
	args.BlobHashes = []common.Hash{{0: 1}}
	args.BlobFeeCap = (*hexutil.Big)(big.NewInt(1))
	result, err := api.Call(context.Background(), args, Outputs{"stateDiff"}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	// The sender pays gasUsed times the price plus blob gas times the blob
	// base fee (1 at zero excess blob gas).
	change := result.StateDiff[testSender].Balance.(map[string]any)["*"].(map[string]any)
	paid := new(big.Int).Sub(change["from"].(*hexutil.Big).ToInt(), change["to"].(*hexutil.Big).ToInt())
	want := new(big.Int).Add(new(big.Int).SetUint64(params.TxGas*params.InitialBaseFee), new(big.Int).SetUint64(params.BlobTxBlobGasPerBlob))
	if paid.Cmp(want) != 0 {
		t.Fatalf("sender paid %v, want %v", paid, want)
	}
}

// pendingBackend serves a pending block and post-state as the miner does.
type pendingBackend struct {
	*testBackend
	block *types.Block
	state *state.StateDB
}

func (b pendingBackend) Pending() (*types.Block, types.Receipts, *state.StateDB) {
	if b.state == nil {
		return b.block, nil, nil
	}
	return b.block, nil, b.state.Copy()
}

func TestPending(t *testing.T) {
	key, _ := crypto.HexToECDSA(strings.Repeat("0", 63) + "5")
	sender := crypto.PubkeyToAddress(key.PublicKey)
	payee := common.HexToAddress("0xcafe0005")
	number, balance, counter := common.HexToAddress("0xcafe0006"), common.HexToAddress("0xcafe0007"), common.HexToAddress("0xcafe0008")
	config := *params.AllEthashProtocolChanges
	genesis := func() *core.Genesis {
		return &core.Genesis{Config: &config, GasLimit: 30_000_000, BaseFee: big.NewInt(params.InitialBaseFee), Alloc: types.GenesisAlloc{
			sender: {Balance: big.NewInt(1e18)},
			// Return NUMBER, and the payee's BALANCE.
			number:  {Code: common.FromHex("4360005260206000f3")},
			balance: {Code: append(append(common.FromHex("73"), payee.Bytes()...), common.FromHex("3160005260206000f3")...)},
			// Increment slot 0 and return the new value.
			counter: {Code: common.FromHex("6000546001018060005560005260206000f3")},
		}}
	}
	// Each block pays the payee one wei. The node has mined the first block;
	// the second, built on it, is pending.
	pay := func(i int, b *core.BlockGen) {
		b.AddTx(types.MustSignNewTx(key, types.LatestSigner(&config), &types.LegacyTx{Nonce: uint64(i), Gas: params.TxGas, GasPrice: big.NewInt(2 * params.InitialBaseFee), To: &payee, Value: common.Big1}))
	}
	backend := newTestBackend(t, 1, genesis(), pay)
	t.Cleanup(backend.teardown)
	next := newTestBackend(t, 2, genesis(), pay)
	t.Cleanup(next.teardown)
	block := next.chain.GetBlockByNumber(2)
	if block.ParentHash() != backend.chain.CurrentBlock().Hash() {
		t.Fatal("pending block does not extend the head")
	}
	st, err := next.chain.StateAt(block.Root(), block.Number(), block.Time())
	if err != nil {
		t.Fatal(err)
	}
	word := func(n int64) hexutil.Bytes { return common.LeftPadBytes(big.NewInt(n).Bytes(), 32) }
	calls := []any{
		[]any{map[string]any{"to": number}, Outputs{}},
		[]any{map[string]any{"to": balance}, Outputs{}},
	}

	// Without a pending block, pending is an invalid parameter for every
	// method rather than an alias of latest.
	client := newTestClient(t, NewAPI(backend))
	var result json.RawMessage
	requireErrorCode(t, client.Call(&result, "trace_block", "pending"), -32602)
	requireErrorCode(t, client.Call(&result, "trace_replayBlockTransactions", "pending", Outputs{"trace"}), -32602)
	requireErrorCode(t, client.Call(&result, "trace_call", calls[0].([]any)[0], Outputs{}, "pending"), -32602)
	requireErrorCode(t, client.Call(&result, "trace_callMany", calls, map[string]any{"blockNumber": "pending"}), -32602)
	// A pending block without its post-state is not a pending environment.
	client = newTestClient(t, NewAPI(pendingBackend{backend, block, nil}))
	requireErrorCode(t, client.Call(&result, "trace_block", "pending"), -32602)
	requireErrorCode(t, client.Call(&result, "trace_call", calls[0].([]any)[0], Outputs{}, "pending"), -32602)

	// With one, block methods trace the pending block and simulations run on
	// its post-state and header: NUMBER is the next block, and the pending
	// transfer is visible.
	client = newTestClient(t, NewAPI(pendingBackend{backend, block, st}))
	var frames []map[string]any
	if err := client.Call(&frames, "trace_block", "pending"); err != nil {
		t.Fatal(err)
	}
	tx := block.Transactions()[0].Hash().Hex()
	// The pending ethash block also carries its block reward.
	if len(frames) != 2 || frames[0]["blockNumber"] != 2.0 || frames[0]["blockHash"] != block.Hash().Hex() || frames[0]["transactionHash"] != tx || frames[1]["type"] != "reward" {
		t.Fatalf("pending block traces: %+v", frames)
	}
	var replays []map[string]any
	if err := client.Call(&replays, "trace_replayBlockTransactions", "pending", Outputs{"trace"}); err != nil {
		t.Fatal(err)
	}
	if len(replays) != 1 || replays[0]["transactionHash"] != tx {
		t.Fatalf("pending block replays: %+v", replays)
	}
	for tag, want := range map[string][]hexutil.Bytes{"latest": {word(1), word(1)}, "pending": {word(2), word(2)}} {
		var many []Execution
		if err := client.Call(&many, "trace_callMany", calls, tag); err != nil {
			t.Fatal(err)
		}
		var one Execution
		if err := client.Call(&one, "trace_call", map[string]any{"to": balance}, Outputs{}, map[string]any{"blockNumber": tag}); err != nil {
			t.Fatal(err)
		}
		if len(many) != 2 || !bytes.Equal(many[0].Output, want[0]) || !bytes.Equal(many[1].Output, want[1]) || !bytes.Equal(one.Output, want[1]) {
			t.Fatalf("%s: NUMBER and BALANCE %+v, %x", tag, many, one.Output)
		}
	}
	// Pending simulations write only their own copy of the pending state:
	// items see earlier writes, later requests start afresh, and overrides
	// do not persist.
	increment := []any{map[string]any{"to": counter}, Outputs{}}
	seven := hexutil.Bytes(common.LeftPadBytes([]byte{7}, 32))
	for _, overrides := range []any{nil, map[string]any{counter.Hex(): map[string]any{"stateDiff": map[string]any{common.Hash{}.Hex(): common.BytesToHash(seven).Hex()}}}} {
		var many []Execution
		if err := client.Call(&many, "trace_callMany", []any{increment, increment}, "pending", overrides); err != nil {
			t.Fatal(err)
		}
		first := word(1)
		if overrides != nil {
			first = word(8)
		}
		if len(many) != 2 || !bytes.Equal(many[0].Output, first) || !bytes.Equal(many[1].Output, common.LeftPadBytes(new(big.Int).Add(new(big.Int).SetBytes(first), common.Big1).Bytes(), 32)) {
			t.Fatalf("pending writes with overrides %v: %+v", overrides, many)
		}
	}
	var fresh Execution
	if err := client.Call(&fresh, "trace_call", increment[0], Outputs{}, "pending"); err != nil || !bytes.Equal(fresh.Output, word(1)) {
		t.Fatalf("pending state leaked between requests: %x %v", fresh.Output, err)
	}
}

// After Amsterdam (EIP-8037), trace_call and trace_callMany may spend more than
// params.MaxTxGas on execution, as eth_call does.
func TestExecutionGasCapAmsterdam(t *testing.T) {
	contract := common.HexToAddress("0x000000000000000000000000000000000000c0de")
	config := *params.MergedTestChainConfig
	config.AmsterdamTime = new(uint64)
	backend := newTestBackend(t, 0, &core.Genesis{Config: &config, Difficulty: common.Big0, GasLimit: 60_000_000, Alloc: types.GenesisAlloc{
		testSender: {Balance: big.NewInt(params.Ether)},
		// Returns the gas left at entry if it is at least 20M, reverts otherwise:
		//   GAS PUSH4 20000000 DUP2 LT PUSH1 0x11 JUMPI
		//   PUSH0 MSTORE PUSH1 0x20 PUSH0 RETURN
		//   JUMPDEST PUSH0 PUSH0 REVERT
		contract: {Code: common.FromHex("0x5a6301312d0081106011575f5260205ff35b5f5ffd")},
	}}, nil)
	t.Cleanup(backend.teardown)
	api := NewAPI(backend)
	args := testCallArgs(&contract, nil)
	gas := hexutil.Uint64(backend.RPCGasCap())
	args.Gas = &gas

	result, err := api.Call(context.Background(), args, Outputs{"trace"}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	results, err := api.CallMany(context.Background(), CallManyArgs{{args, Outputs{"trace"}}}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for method, result := range map[string]*Execution{"trace_call": result, "trace_callMany": results[0]} {
		if failure := result.Trace[0].Error; failure != "" {
			t.Fatalf("%s: execution failed with %q, want execution gas above params.MaxTxGas", method, failure)
		}
	}
}
