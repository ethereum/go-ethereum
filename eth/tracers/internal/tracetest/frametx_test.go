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

package tracetest

import (
	"encoding/json"
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/tests"
	"github.com/holiman/uint256"
)

var (
	frameTxSenderKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	frameTxSender       = crypto.PubkeyToAddress(frameTxSenderKey.PublicKey)
	frameTxPayer        = common.HexToAddress("0x2222222222222222222222222222222222222222")
	frameTxLogger       = common.HexToAddress("0x3333333333333333333333333333333333333333")
	frameTxReverter     = common.HexToAddress("0x4444444444444444444444444444444444444444")
	frameTxNestedCaller = common.HexToAddress("0x5555555555555555555555555555555555555555")
	frameTxCoinbase     = common.HexToAddress("0x00000000000000000000000000000000c0ffee00")
	frameTxEntryPoint   = params.FrameTxEntryPoint
)

func frameTxChainConfig() *params.ChainConfig {
	config := *params.MergedTestChainConfig
	zero := uint64(0)
	config.AmsterdamTime = &zero
	config.BogotaTime = &zero
	return &config
}

func frameTxAlloc() types.GenesisAlloc {
	ether := big.NewInt(params.Ether)
	return types.GenesisAlloc{
		frameTxSender: {Balance: ether},
		frameTxPayer: {
			Balance: ether,
			Code:    program.New().Push(types.ApprovePayment).Push(0).Push(0).Op(vm.APPROVE).Bytes(),
		},
		frameTxLogger: {
			Code: program.New().Push(0xab).Push(0).Push(0).Op(vm.LOG1).Op(vm.STOP).Bytes(),
		},
		frameTxReverter: {
			Code: program.New().Push(0xcd).Push(0).Push(0).Op(vm.LOG1).Push(0).Push(0).Op(vm.REVERT).Bytes(),
		},
		frameTxNestedCaller: {
			Code: program.New().Push(0xef).Push(0).Push(0).Op(vm.LOG1).Call(nil, frameTxLogger, 0, 0, 0, 0, 0).Op(vm.STOP).Bytes(),
		},
	}
}

func verifySenderFrame(flags uint64) types.Frame {
	return types.Frame{Mode: types.ModeVerify, Flags: flags, GasLimits: types.Limits{Execution: 100_000}, Value: new(uint256.Int)}
}

func verifyPayerFrame() types.Frame {
	return types.Frame{Mode: types.ModeVerify, Flags: types.ApprovePayment, Target: &frameTxPayer, GasLimits: types.Limits{Execution: 100_000}, Value: new(uint256.Int)}
}

func senderFrame(target common.Address, flags uint64, value *uint256.Int) types.Frame {
	return types.Frame{Mode: types.ModeSender, Flags: flags, Target: &target, GasLimits: types.Limits{Execution: 200_000, State: 50_000}, Value: value, Data: []byte{0xa9, 0x05, 0x9c, 0xbb}}
}

func newSignedFrameTx(t *testing.T, config *params.ChainConfig, frames []types.Frame) *types.Transaction {
	t.Helper()
	frameTx := &types.FrameTx{
		ChainID: uint256.MustFromBig(config.ChainID),
		Sender:  frameTxSender,
		Frames:  frames,
		Signatures: types.SignatureList{{
			Scheme: types.FrameTxSchemeSecp256k1,
			Signer: frameTxSender.Bytes(),
		}},
		Fees: types.Fees{
			MaxPriorityFeePerGas: uint256.NewInt(1),
			MaxFeePerGas:         uint256.NewInt(params.GWei),
			MaxFeePerBlobGas:     uint256.NewInt(0),
		},
	}
	sigHash := types.LatestSigner(config).Hash(types.NewTx(frameTx))
	sig, err := crypto.Sign(sigHash[:], frameTxSenderKey)
	if err != nil {
		t.Fatalf("failed to sign frame transaction: %v", err)
	}
	frameTx.Signatures[0].Signature = append([]byte{sig[64]}, sig[:64]...)
	return types.NewTx(frameTx)
}

func traceFrameTx(t *testing.T, frames []types.Frame, tracerName string, tracerConfig string) (json.RawMessage, *types.Transaction, *types.Receipt) {
	t.Helper()
	var (
		config  = frameTxChainConfig()
		tx      = newSignedFrameTx(t, config, frames)
		signer  = types.LatestSigner(config)
		baseFee = big.NewInt(7)
		context = vm.BlockContext{
			CanTransfer:      core.CanTransfer,
			Transfer:         core.Transfer,
			GetHash:          func(uint64) common.Hash { return common.Hash{} },
			Coinbase:         frameTxCoinbase,
			BlockNumber:      big.NewInt(1),
			Time:             1,
			Difficulty:       new(big.Int),
			BaseFee:          baseFee,
			BlobBaseFee:      big.NewInt(1),
			GasLimit:         30_000_000,
			Random:           &common.Hash{},
			CostPerStateByte: params.CostPerStateByte,
		}
	)
	st := tests.MakePreState(rawdb.NewMemoryDatabase(), frameTxAlloc(), false, rawdb.HashScheme)
	defer st.Close()

	tracer, err := tracers.DefaultDirectory.New(tracerName, &tracers.Context{TxHash: tx.Hash()}, json.RawMessage(tracerConfig), config)
	if err != nil {
		t.Fatalf("failed to create tracer: %v", err)
	}
	evm := vm.NewEVM(context, state.NewHookedState(st.StateDB, tracer.Hooks), config, vm.Config{Tracer: tracer.Hooks})
	msg, err := core.TransactionToMessage(tx, signer, baseFee)
	if err != nil {
		t.Fatalf("failed to create message: %v", err)
	}
	st.StateDB.SetTxContext(tx.Hash(), 0, 1)
	receipt, _, err := core.ApplyTransactionWithEVM(msg, core.NewGasPool(context.GasLimit), st.StateDB, context.BlockNumber, common.Hash{}, context.Time, tx, evm)
	if err != nil {
		t.Fatalf("failed to apply frame transaction: %v", err)
	}
	res, err := tracer.GetResult()
	if err != nil {
		t.Fatalf("failed to retrieve trace result: %v", err)
	}
	return res, tx, receipt
}

type frameTraceCall struct {
	Type    string           `json:"type"`
	From    common.Address   `json:"from"`
	To      common.Address   `json:"to"`
	Value   *hexutil.Big     `json:"value"`
	Gas     hexutil.Uint64   `json:"gas"`
	GasUsed hexutil.Uint64   `json:"gasUsed"`
	Input   hexutil.Bytes    `json:"input"`
	Error   string           `json:"error"`
	Calls   []frameTraceCall `json:"calls"`
	Logs    []frameTraceLog  `json:"logs"`
}

type frameTraceLog struct {
	Address common.Address `json:"address"`
	Topics  []common.Hash  `json:"topics"`
}

func (c *frameTraceCall) subtreeLogs() []frameTraceLog {
	logs := slices.Clone(c.Logs)
	for i := range c.Calls {
		logs = append(logs, c.Calls[i].subtreeLogs()...)
	}
	return logs
}

type expectedFrameCall struct {
	err         string
	nestedCalls int
}

func TestFrameTxCallTracer(t *testing.T) {
	var (
		sponsored = []types.Frame{
			verifySenderFrame(types.ApproveExecution),
			verifyPayerFrame(),
			senderFrame(frameTxNestedCaller, 0, new(uint256.Int)),
		}
		senderRevert = []types.Frame{
			verifySenderFrame(types.ApproveExecutionAndPayment),
			senderFrame(frameTxReverter, 0, new(uint256.Int)),
		}
		atomicBatchFailure = []types.Frame{
			verifySenderFrame(types.ApproveExecutionAndPayment),
			senderFrame(frameTxLogger, types.AtomicBatchFlag, new(uint256.Int)),
			senderFrame(frameTxReverter, types.AtomicBatchFlag, new(uint256.Int)),
			senderFrame(frameTxLogger, 0, new(uint256.Int)),
			senderFrame(frameTxLogger, 0, new(uint256.Int)),
		}
		preEVMFailure = []types.Frame{
			verifySenderFrame(types.ApproveExecutionAndPayment),
			senderFrame(frameTxLogger, 0, uint256.NewInt(2*params.Ether)),
			senderFrame(frameTxLogger, 0, new(uint256.Int)),
		}
	)
	for _, tc := range []struct {
		name        string
		frames      []types.Frame
		config      string
		wantRootErr bool
		want        []expectedFrameCall
	}{
		{
			name:   "sponsored",
			frames: sponsored,
			config: `{"withLog":true}`,
			want:   []expectedFrameCall{{}, {}, {nestedCalls: 1}},
		},
		{
			name:   "sponsored onlyTopCall",
			frames: sponsored,
			config: `{"onlyTopCall":true,"withLog":true}`,
			want:   []expectedFrameCall{{}, {}, {}},
		},
		{
			name:        "sender frame revert",
			frames:      senderRevert,
			config:      `{"withLog":true}`,
			wantRootErr: true,
			want:        []expectedFrameCall{{}, {err: "execution reverted"}},
		},
		{
			name:        "atomic batch failure",
			frames:      atomicBatchFailure,
			config:      `{"withLog":true}`,
			wantRootErr: true,
			want:        []expectedFrameCall{{}, {}, {err: "execution reverted"}, {err: "frame skipped"}, {}},
		},
		{
			name:        "pre-EVM failure",
			frames:      preEVMFailure,
			config:      `{"withLog":true}`,
			wantRootErr: true,
			want:        []expectedFrameCall{{}, {err: "insufficient balance for transfer"}, {}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, tx, receipt := traceFrameTx(t, tc.frames, "callTracer", tc.config)
			var root frameTraceCall
			if err := json.Unmarshal(res, &root); err != nil {
				t.Fatalf("failed to decode trace: %v", err)
			}
			if root.Type != "CALL" || root.From != frameTxSender || root.To != frameTxEntryPoint {
				t.Errorf("root call: have %s %v -> %v, want CALL %v -> %v", root.Type, root.From, root.To, frameTxSender, frameTxEntryPoint)
			}
			if root.Value == nil || root.Value.ToInt().Sign() != 0 || len(root.Input) != 0 {
				t.Errorf("root value/input: have %v/%x, want 0x0/0x", root.Value, root.Input)
			}
			if uint64(root.Gas) != tx.Gas() || uint64(root.GasUsed) != receipt.GasUsed {
				t.Errorf("root gas: have %d/%d, want %d/%d", root.Gas, root.GasUsed, tx.Gas(), receipt.GasUsed)
			}
			if (root.Error != "") != tc.wantRootErr {
				t.Errorf("root error: have %q, want error %v", root.Error, tc.wantRootErr)
			}
			if len(root.Logs) != 0 {
				t.Errorf("root logs: have %d, want 0", len(root.Logs))
			}
			if len(root.Calls) != len(tc.frames) {
				t.Fatalf("frame calls: have %d, want %d", len(root.Calls), len(tc.frames))
			}
			for i, frame := range tc.frames {
				var (
					call         = root.Calls[i]
					frameReceipt = receipt.FrameReceipts[i]
					wantType     = "CALL"
					wantFrom     = frameTxEntryPoint
				)
				if frame.Mode == types.ModeVerify {
					wantType = "STATICCALL"
				}
				if frame.Mode == types.ModeSender {
					wantFrom = frameTxSender
				}
				if call.Type != wantType || call.From != wantFrom || call.To != frame.ResolvedTarget(frameTxSender) {
					t.Errorf("frame %d: have %s %v -> %v, want %s %v -> %v", i, call.Type, call.From, call.To, wantType, wantFrom, frame.ResolvedTarget(frameTxSender))
				}
				if frame.Mode != types.ModeVerify && (call.Value == nil || call.Value.ToInt().Cmp(frame.Value.ToBig()) != 0) {
					t.Errorf("frame %d value: have %v, want %v", i, call.Value, frame.Value)
				}
				if !slices.Equal(call.Input, frame.Data) {
					t.Errorf("frame %d input: have %x, want %x", i, call.Input, frame.Data)
				}
				if uint64(call.Gas) != frame.GasLimits.Execution+frame.GasLimits.State || uint64(call.GasUsed) != frameReceipt.GasUsed {
					t.Errorf("frame %d gas: have %d/%d, want %d/%d", i, call.Gas, call.GasUsed, frame.GasLimits.Execution+frame.GasLimits.State, frameReceipt.GasUsed)
				}
				if call.Error != tc.want[i].err {
					t.Errorf("frame %d error: have %q, want %q", i, call.Error, tc.want[i].err)
				}
				if (call.Error != "") != (frameReceipt.Status != types.ReceiptStatusSuccessful) {
					t.Errorf("frame %d error %q disagrees with receipt status %d", i, call.Error, frameReceipt.Status)
				}
				if len(call.Calls) != tc.want[i].nestedCalls {
					t.Errorf("frame %d nested calls: have %d, want %d", i, len(call.Calls), tc.want[i].nestedCalls)
				}
				if tc.name == "sponsored onlyTopCall" {
					continue
				}
				logs := call.subtreeLogs()
				if len(logs) != len(frameReceipt.Logs) {
					t.Fatalf("frame %d logs: have %d, want %d", i, len(logs), len(frameReceipt.Logs))
				}
				for j := range logs {
					if logs[j].Address != frameReceipt.Logs[j].Address || !slices.Equal(logs[j].Topics, frameReceipt.Logs[j].Topics) {
						t.Errorf("frame %d log %d: have %v %v, want %v %v", i, j, logs[j].Address, logs[j].Topics, frameReceipt.Logs[j].Address, frameReceipt.Logs[j].Topics)
					}
				}
			}
		})
	}
}

func TestFrameTxCallTracerLogsFollowFrameReceipts(t *testing.T) {
	frames := []types.Frame{
		verifySenderFrame(types.ApproveExecutionAndPayment),
		senderFrame(frameTxLogger, types.AtomicBatchFlag, new(uint256.Int)),
		senderFrame(frameTxReverter, 0, new(uint256.Int)),
		senderFrame(frameTxNestedCaller, 0, new(uint256.Int)),
	}
	res, _, receipt := traceFrameTx(t, frames, "callTracer", `{"withLog":true}`)
	var root frameTraceCall
	if err := json.Unmarshal(res, &root); err != nil {
		t.Fatalf("failed to decode trace: %v", err)
	}
	wantLogCounts := []int{0, 0, 0, 2}
	for i, want := range wantLogCounts {
		if have := len(receipt.FrameReceipts[i].Logs); have != want {
			t.Fatalf("frame %d receipt logs: have %d, want %d", i, have, want)
		}
		if have := len(root.Calls[i].subtreeLogs()); have != want {
			t.Errorf("frame %d trace logs: have %d, want %d", i, have, want)
		}
	}
	if root.Calls[1].Error != "" || receipt.FrameReceipts[1].Status != types.ReceiptStatusSuccessful {
		t.Errorf("rolled back batch frame: have error %q status %d, want no error status 1", root.Calls[1].Error, receipt.FrameReceipts[1].Status)
	}
}

type frameFlatTrace struct {
	Action struct {
		CallType string         `json:"callType"`
		From     common.Address `json:"from"`
		To       common.Address `json:"to"`
		Gas      hexutil.Uint64 `json:"gas"`
		Value    *hexutil.Big   `json:"value"`
		Input    hexutil.Bytes  `json:"input"`
	} `json:"action"`
	Error  string `json:"error"`
	Result *struct {
		GasUsed hexutil.Uint64 `json:"gasUsed"`
	} `json:"result"`
	Subtraces    int    `json:"subtraces"`
	TraceAddress []int  `json:"traceAddress"`
	Type         string `json:"type"`
}

func TestFrameTxFlatCallTracer(t *testing.T) {
	frames := []types.Frame{
		verifySenderFrame(types.ApproveExecution),
		verifyPayerFrame(),
		senderFrame(frameTxNestedCaller, types.AtomicBatchFlag, new(uint256.Int)),
		senderFrame(frameTxReverter, types.AtomicBatchFlag, new(uint256.Int)),
		senderFrame(frameTxLogger, 0, new(uint256.Int)),
		senderFrame(frameTxLogger, 0, uint256.NewInt(2*params.Ether)),
	}
	res, tx, receipt := traceFrameTx(t, frames, "flatCallTracer", `{"convertParityErrors":true}`)
	var traces []frameFlatTrace
	if err := json.Unmarshal(res, &traces); err != nil {
		t.Fatalf("failed to decode trace: %v", err)
	}
	if len(traces) != 1+len(frames)+1 {
		t.Fatalf("trace count: have %d, want %d", len(traces), 1+len(frames)+1)
	}
	root := traces[0]
	if root.Type != "call" || root.Action.CallType != "call" || len(root.TraceAddress) != 0 || root.TraceAddress == nil {
		t.Errorf("root: have type %s/%s traceAddress %v", root.Type, root.Action.CallType, root.TraceAddress)
	}
	if root.Action.From != frameTxSender || root.Action.To != frameTxEntryPoint || root.Action.Value.ToInt().Sign() != 0 || len(root.Action.Input) != 0 {
		t.Errorf("root action: have %v -> %v value %v input %x", root.Action.From, root.Action.To, root.Action.Value, root.Action.Input)
	}
	if uint64(root.Action.Gas) != tx.Gas() || root.Result == nil || uint64(root.Result.GasUsed) != receipt.GasUsed || root.Subtraces != len(frames) {
		t.Errorf("root gas/subtraces: have %d/%v/%d, want %d/%d/%d", root.Action.Gas, root.Result, root.Subtraces, tx.Gas(), receipt.GasUsed, len(frames))
	}
	var (
		wantCallTypes = []string{"staticcall", "staticcall", "call", "call", "call", "call"}
		wantErrors    = []string{"", "", "", "Reverted", "frame skipped", "insufficient balance for transfer"}
		wantResult    = []bool{true, true, true, true, false, false}
		wantSubtraces = []int{0, 0, 1, 0, 0, 0}
	)
	frameTraces := slices.DeleteFunc(slices.Clone(traces[1:]), func(tr frameFlatTrace) bool { return len(tr.TraceAddress) != 1 })
	if len(frameTraces) != len(frames) {
		t.Fatalf("frame traces: have %d, want %d", len(frameTraces), len(frames))
	}
	for i, tr := range frameTraces {
		frame := frames[i]
		if tr.TraceAddress[0] != i || tr.Action.CallType != wantCallTypes[i] {
			t.Errorf("frame %d: have traceAddress %v callType %s, want [%d] %s", i, tr.TraceAddress, tr.Action.CallType, i, wantCallTypes[i])
		}
		if tr.Action.To != frame.ResolvedTarget(frameTxSender) || uint64(tr.Action.Gas) != frame.GasLimits.Execution+frame.GasLimits.State {
			t.Errorf("frame %d action: have to %v gas %d", i, tr.Action.To, tr.Action.Gas)
		}
		if tr.Action.Value == nil || tr.Action.Value.ToInt().Cmp(frame.Value.ToBig()) != 0 {
			t.Errorf("frame %d value: have %v, want %v", i, tr.Action.Value, frame.Value)
		}
		if tr.Error != wantErrors[i] {
			t.Errorf("frame %d error: have %q, want %q", i, tr.Error, wantErrors[i])
		}
		if (tr.Result != nil) != wantResult[i] {
			t.Errorf("frame %d result: have %v, want present %v", i, tr.Result, wantResult[i])
		}
		if tr.Result != nil && uint64(tr.Result.GasUsed) != receipt.FrameReceipts[i].GasUsed {
			t.Errorf("frame %d gasUsed: have %d, want %d", i, tr.Result.GasUsed, receipt.FrameReceipts[i].GasUsed)
		}
		if tr.Subtraces != wantSubtraces[i] {
			t.Errorf("frame %d subtraces: have %d, want %d", i, tr.Subtraces, wantSubtraces[i])
		}
	}
	nested := traces[4]
	if !slices.Equal(nested.TraceAddress, []int{2, 0}) || nested.Action.From != frameTxNestedCaller || nested.Action.To != frameTxLogger {
		t.Errorf("nested call: have traceAddress %v %v -> %v", nested.TraceAddress, nested.Action.From, nested.Action.To)
	}
}

func TestFrameTxPrestateTracer(t *testing.T) {
	frames := []types.Frame{
		verifySenderFrame(types.ApproveExecution),
		verifyPayerFrame(),
		senderFrame(frameTxNestedCaller, 0, new(uint256.Int)),
	}
	for _, config := range []string{`{}`, `{"includeEmpty":true}`, `{"diffMode":true}`} {
		t.Run(config, func(t *testing.T) {
			res, tx, receipt := traceFrameTx(t, frames, "prestateTracer", config)
			var result struct {
				Pre map[common.Address]json.RawMessage `json:"pre"`
			}
			if config != `{"diffMode":true}` {
				if err := json.Unmarshal(res, &result.Pre); err != nil {
					t.Fatalf("failed to decode trace: %v", err)
				}
			} else if err := json.Unmarshal(res, &result); err != nil {
				t.Fatalf("failed to decode trace: %v", err)
			}
			if receipt.Payer == nil || *receipt.Payer != frameTxPayer {
				t.Fatalf("receipt payer: have %v, want %v", receipt.Payer, frameTxPayer)
			}
			want := []common.Address{frameTxSender, frameTxPayer}
			if config != `{"diffMode":true}` {
				want = append(want, frameTxNestedCaller, frameTxLogger)
			}
			for _, addr := range want {
				if _, ok := result.Pre[addr]; !ok {
					t.Errorf("prestate misses %v", addr)
				}
			}
			phantom := crypto.CreateAddress(frameTxSender, tx.Nonce())
			if _, ok := result.Pre[phantom]; ok {
				t.Errorf("prestate includes phantom create address %v", phantom)
			}
			if _, ok := result.Pre[frameTxEntryPoint]; ok {
				t.Errorf("prestate includes untouched entry point %v", frameTxEntryPoint)
			}
			var payer struct {
				Balance *hexutil.Big `json:"balance"`
			}
			if err := json.Unmarshal(result.Pre[frameTxPayer], &payer); err != nil {
				t.Fatalf("failed to decode payer: %v", err)
			}
			if payer.Balance == nil || payer.Balance.ToInt().Cmp(big.NewInt(params.Ether)) != 0 {
				t.Errorf("payer prestate balance: have %v, want %v", payer.Balance, big.NewInt(params.Ether))
			}
		})
	}
}
