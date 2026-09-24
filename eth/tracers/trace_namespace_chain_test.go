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
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

func TestTraceNamespaceMinedPathsReplayAndFilter(t *testing.T) {
	key, _ := crypto.HexToECDSA(strings.Repeat("0", 63) + "4")
	sender := crypto.PubkeyToAddress(key.PublicKey)
	child := common.HexToAddress("0xcafe0003")
	miner := common.HexToAddress("0xcafe0004")
	childCode := common.FromHex("602a60005260206000f3")
	parentCode := append(common.FromHex("6000600060006000600073"), child.Bytes()...)
	parentCode = append(parentCode, common.FromHex("61fffff15000")...)
	config := *params.AllEthashProtocolChanges
	genesis := &core.Genesis{Config: &config, GasLimit: 30_000_000, BaseFee: big.NewInt(params.InitialBaseFee), Alloc: types.GenesisAlloc{
		sender:          {Balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil)},
		traceTestTarget: {Balance: big.NewInt(0), Code: parentCode}, child: {Balance: big.NewInt(0), Code: childCode},
	}}
	var hashes []common.Hash
	backend := newTestBackend(t, 2, genesis, func(i int, b *core.BlockGen) {
		b.SetCoinbase(miner)
		tx := types.MustSignNewTx(key, types.LatestSigner(&config), &types.LegacyTx{Nonce: uint64(i), Gas: 200000, GasPrice: big.NewInt(2 * params.InitialBaseFee), To: &traceTestTarget})
		b.AddTx(tx)
		hashes = append(hashes, tx.Hash())
	})
	defer backend.teardown()
	api := NewTraceAPI(backend)
	ctx := context.Background()
	root, err := api.Get(ctx, hashes[0], TracePosition{})
	if err != nil {
		t.Fatal(err)
	}
	if root == nil || len(root.TraceAddress) != 0 || root.Subtraces != 1 {
		t.Fatalf("root: %+v", root)
	}
	nested, err := api.Get(ctx, hashes[0], TracePosition{0})
	if err != nil {
		t.Fatal(err)
	}
	if nested == nil || nested.Action.(traceCallAction).To != child {
		t.Fatalf("child: %+v", nested)
	}
	missing, err := api.Get(ctx, hashes[0], TracePosition{0, 0})
	if err != nil || missing != nil {
		t.Fatalf("missing: %+v %v", missing, err)
	}
	replay, err := api.ReplayTransaction(ctx, hashes[0], TraceTypes{"trace", "vmTrace", "stateDiff"})
	if err != nil {
		t.Fatal(err)
	}
	if replay.TransactionHash == nil || *replay.TransactionHash != hashes[0] {
		t.Fatal("replay identity missing")
	}
	if len(replay.Trace) != 2 || replay.Trace[0].traceLocation != nil {
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
	blockReplay, err := api.ReplayBlockTransactions(ctx, 1, TraceTypes{"trace"})
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
	if traces[2].Action.(traceRewardAction).Value.ToInt().Cmp(big.NewInt(2e18)) != 0 {
		t.Fatal("wrong PoW issuance")
	}
	from, to := rpc.BlockNumber(1), rpc.BlockNumber(2)
	filter := TraceFilter{FromBlock: &from, ToBlock: &to, FromAddress: []common.Address{sender}, ToAddress: []common.Address{child}}
	frames, err := api.Filter(ctx, filter)
	if err != nil || len(frames) != 0 {
		t.Fatalf("filters must intersect: %+v %v", frames, err)
	}
	intersection := len(frames)
	filter.Mode = TraceFilterUnion
	frames, err = api.Filter(ctx, filter)
	if err != nil || len(frames) <= intersection {
		t.Fatalf("union must widen the intersection: %+v %v", frames, err)
	}
	for _, frame := range frames {
		if !traceMatches(frame, TraceFilter{FromAddress: filter.FromAddress}) && !traceMatches(frame, TraceFilter{ToAddress: filter.ToAddress}) {
			t.Fatalf("union returned a frame matching neither list: %+v", frame)
		}
	}
	// A one-sided union selects the same frames as the default.
	oneSided := TraceFilter{FromBlock: &from, ToBlock: &to, FromAddress: filter.FromAddress}
	plain, err := api.Filter(ctx, oneSided)
	if err != nil {
		t.Fatal(err)
	}
	oneSided.Mode = TraceFilterUnion
	if union, err := api.Filter(ctx, oneSided); err != nil || len(union) != len(plain) || len(plain) == 0 {
		t.Fatalf("one-sided union: %d vs %d %v", len(union), len(plain), err)
	}
	filter.Mode = ""
	filter.FromAddress = []common.Address{traceTestTarget}
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
	filter = TraceFilter{FromBlock: &from, ToBlock: &to, ToAddress: []common.Address{miner}}
	frames, err = api.Filter(ctx, filter)
	if err != nil || len(frames) != 2 || frames[0].Type != "reward" {
		t.Fatalf("reward filtering: %+v %v", frames, err)
	}
}

func TestTraceNamespaceSelfDestructForkRules(t *testing.T) {
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
			traceTestSender: {Balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil)}, traceTestTarget: {Balance: big.NewInt(7), Code: code, Storage: map[common.Hash]common.Hash{{}: {31: 7}}},
		}}, nil)
		api := NewTraceAPI(backend)
		result, err := api.Call(context.Background(), traceTestArgs(&traceTestTarget, nil), TraceTypes{"trace", "stateDiff", "vmTrace"}, nil, nil, nil)
		backend.teardown()
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Trace) != 2 || result.Trace[1].Type != "suicide" {
			t.Fatalf("selfdestruct missing: %+v", result.Trace)
		}
		diff := result.StateDiff[traceTestTarget]
		if diff == nil {
			t.Fatal("missing account transition")
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

func TestTraceNamespaceTouchedEmptyAccountDeletion(t *testing.T) {
	empty := common.HexToAddress("0xcafe0006")
	// CALL the empty account without value.
	code := append(common.FromHex("600060006000600060007"+"3"), empty.Bytes()...)
	code = append(code, common.FromHex("61fffff15000")...)
	config := *params.AllDevChainProtocolChanges
	// Genesis keeps EIP-161-empty accounts, including the zero-address coinbase.
	backend := newTestBackend(t, 0, &core.Genesis{Config: &config, GasLimit: 30_000_000, Difficulty: big.NewInt(0), BaseFee: big.NewInt(params.InitialBaseFee), Alloc: types.GenesisAlloc{
		traceTestSender: {Balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil)}, traceTestTarget: {Code: code},
		empty: {Balance: common.Big0}, common.Address{}: {Balance: common.Big0},
	}}, nil)
	t.Cleanup(backend.teardown)
	api := NewTraceAPI(backend)
	deleted := `{"balance":{"-":"0x0"},"nonce":{"-":"0x0"},"code":{"-":"0x"},"storage":{}}`
	// A zero-fee call skips the fee payment and does not touch the coinbase.
	result, err := api.Call(context.Background(), traceTestArgs(&traceTestTarget, nil), TraceTypes{"stateDiff"}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if have, _ := json.Marshal(result.StateDiff[empty]); string(have) != deleted {
		t.Fatalf("touched empty account: have %s, want %s", have, deleted)
	}
	if diff := result.StateDiff[common.Address{}]; diff != nil {
		t.Fatalf("untouched coinbase: %+v", diff)
	}
	// Paying a zero tip touches the empty coinbase, which is then deleted.
	args := traceTestArgs(&traceTestTarget, nil)
	args.GasPrice = (*hexutil.Big)(big.NewInt(params.InitialBaseFee))
	if result, err = api.Call(context.Background(), args, TraceTypes{"stateDiff"}, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	for _, address := range []common.Address{empty, {}} {
		if have, _ := json.Marshal(result.StateDiff[address]); string(have) != deleted {
			t.Fatalf("%x: have %s, want %s", address, have, deleted)
		}
	}
}

func TestTraceNamespaceBlobFeeAccounting(t *testing.T) {
	api, _ := traceTestAPI(t, common.FromHex("00"), nil)
	args := traceTestArgs(&traceTestTarget, nil)
	args.MaxFeePerGas = (*hexutil.Big)(big.NewInt(params.InitialBaseFee))
	args.BlobHashes = []common.Hash{{0: 1}}
	args.BlobFeeCap = (*hexutil.Big)(big.NewInt(1))
	result, err := api.Call(context.Background(), args, TraceTypes{"stateDiff"}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	// The sender pays gasUsed times the price plus blob gas times the blob
	// base fee (1 at zero excess blob gas).
	change := result.StateDiff[traceTestSender].Balance.(map[string]any)["*"].(map[string]any)
	paid := new(big.Int).Sub(change["from"].(*hexutil.Big).ToInt(), change["to"].(*hexutil.Big).ToInt())
	want := new(big.Int).Add(new(big.Int).SetUint64(params.TxGas*params.InitialBaseFee), new(big.Int).SetUint64(params.BlobTxBlobGasPerBlob))
	if paid.Cmp(want) != 0 {
		t.Fatalf("sender paid %v, want %v", paid, want)
	}
}
