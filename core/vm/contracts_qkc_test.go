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

package vm

import (
	"bytes"
	"errors"
	"math/big"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

const xShardOutboxABIJSON = `[
	{"type": "function", "name": "send", "stateMutability": "payable", "outputs": [], "inputs": [
		{"name": "dstShard", "type": "uint32"},
		{"name": "to", "type": "address"},
		{"name": "value", "type": "uint256"},
		{"name": "gasLimit", "type": "uint64"},
		{"name": "data", "type": "bytes"}
	]},
	{"type": "event", "name": "SentMessage", "inputs": [
		{"name": "dstShard", "type": "uint32", "indexed": true},
		{"name": "from", "type": "address", "indexed": true},
		{"name": "to", "type": "address", "indexed": true},
		{"name": "value", "type": "uint256"},
		{"name": "gasPrice", "type": "uint256"},
		{"name": "gasLimit", "type": "uint64"},
		{"name": "data", "type": "bytes"}
	]}
]`

var (
	xShardOutboxABI, _ = abi.JSON(strings.NewReader(xShardOutboxABIJSON))

	xShardSender    = common.HexToAddress("0x1000000000000000000000000000000000000001")
	xShardRecipient = common.HexToAddress("0x2000000000000000000000000000000000000002")
	xShardFunds     = uint256.NewInt(1e18)
)

const (
	xShardGasPrice = 10
	xShardValue    = 1000
	xShardGasLimit = 50000
	xShardDst      = 2
)

func xShardChainConfig() *params.ChainConfig {
	cfg := *params.MergedTestChainConfig
	cfg.QuarkChain = &params.QuarkChainConfig{ShardID: 1, ShardIDs: []uint32{0, 1, 2}}
	return &cfg
}

// newXShardEVM returns an EVM of shard 1 of a network of shards 0, 1 and 2,
// running a transaction of xShardSender at xShardGasPrice.
func newXShardEVM() (*EVM, *state.StateDB) {
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	statedb.AddBalance(xShardSender, xShardFunds, tracing.BalanceChangeUnspecified)
	ctx := BlockContext{
		CanTransfer: func(db StateDB, addr common.Address, amount *uint256.Int) bool {
			return db.GetBalance(addr).Cmp(amount) >= 0
		},
		Transfer: func(db StateDB, sender, recipient common.Address, amount *uint256.Int, _ *params.Rules) {
			db.SubBalance(sender, amount, tracing.BalanceChangeTransfer)
			db.AddBalance(recipient, amount, tracing.BalanceChangeTransfer)
		},
		BlockNumber: big.NewInt(0),
		Random:      &common.Hash{},
	}
	evm := NewEVM(ctx, statedb, xShardChainConfig(), Config{})
	evm.SetTxContext(TxContext{Origin: xShardSender, GasPrice: uint256.NewInt(xShardGasPrice)})
	return evm, statedb
}

func packXShardSend(t *testing.T, dstShard uint32, to common.Address, value *big.Int, gasLimit uint64, data []byte) []byte {
	t.Helper()
	input, err := xShardOutboxABI.Pack("send", dstShard, to, value, gasLimit, data)
	if err != nil {
		t.Fatalf("failed to pack input: %v", err)
	}
	return input
}

func TestXShardOutboxSend(t *testing.T) {
	for _, tt := range []struct {
		name     string
		data     []byte
		excess   uint64
		gasLimit uint64 // defaults to xShardGasLimit
	}{
		{name: "no data"},
		{name: "data", data: []byte{0xde, 0xad, 0xbe, 0xef}},
		{name: "word sized data", data: bytes.Repeat([]byte{0xaa}, 64)},
		{name: "excess value", data: []byte{1, 2, 3}, excess: 12345},
		{name: "minimum gas limit", gasLimit: params.XShardSendGas},
		{name: "maximum gas limit", gasLimit: params.MaxTxGas},
	} {
		t.Run(tt.name, func(t *testing.T) {
			evm, statedb := newXShardEVM()
			gasLimit := tt.gasLimit
			if gasLimit == 0 {
				gasLimit = xShardGasLimit
			}
			var (
				input    = packXShardSend(t, xShardDst, xShardRecipient, big.NewInt(xShardValue), gasLimit, tt.data)
				cost     = uint256.NewInt(xShardValue + gasLimit*xShardGasPrice)
				callVal  = new(uint256.Int).AddUint64(cost, tt.excess)
				gas      = NewGasBudget(1_000_000, 0)
				wantUsed = (&xShardOutbox{}).RequiredGas(input)
			)
			ret, left, err := evm.Call(xShardSender, params.XShardOutboxAddress, input, gas, callVal)
			if err != nil {
				t.Fatalf("send failed: %v", err)
			}
			if len(ret) != 0 {
				t.Errorf("unexpected return data %x", ret)
			}
			if used := left.Used(gas); used != wantUsed {
				t.Errorf("gas used %d, want %d", used, wantUsed)
			}
			// Exactly the cost is burnt, the excess goes back to the sender.
			if have, want := statedb.GetBalance(xShardSender), new(uint256.Int).Sub(xShardFunds, cost); !have.Eq(want) {
				t.Errorf("sender balance %v, want %v", have, want)
			}
			if have := statedb.GetBalance(params.XShardOutboxAddress); !have.IsZero() {
				t.Errorf("outbox balance %v, want 0", have)
			}
			logs := statedb.Logs()
			if len(logs) != 1 {
				t.Fatalf("got %d logs, want 1", len(logs))
			}
			checkSentMessageLog(t, logs[0], gasLimit, tt.data)
		})
	}
}

// checkSentMessageLog checks a SentMessage log of the default test message
// with the given gas limit and data, decoding it with the standard ABI decoder.
func checkSentMessageLog(t *testing.T, log *types.Log, gasLimit uint64, data []byte) {
	t.Helper()
	if log.Address != params.XShardOutboxAddress {
		t.Errorf("log address %v, want %v", log.Address, params.XShardOutboxAddress)
	}
	event := xShardOutboxABI.Events["SentMessage"]
	if len(log.Topics) != 4 || log.Topics[0] != event.ID || event.ID != XShardSentMessageTopic {
		t.Fatalf("log topics %v, want SentMessage topic %v", log.Topics, event.ID)
	}
	var indexed abi.Arguments
	for _, arg := range event.Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	fields := make(map[string]any)
	if err := abi.ParseTopicsIntoMap(fields, indexed, log.Topics[1:]); err != nil {
		t.Fatalf("failed to parse topics: %v", err)
	}
	if err := xShardOutboxABI.UnpackIntoMap(fields, "SentMessage", log.Data); err != nil {
		t.Fatalf("failed to unpack data: %v", err)
	}
	want := map[string]any{
		"dstShard": uint32(xShardDst),
		"from":     xShardSender,
		"to":       xShardRecipient,
		"value":    big.NewInt(xShardValue),
		"gasPrice": big.NewInt(xShardGasPrice),
		"gasLimit": gasLimit,
		"data":     data,
	}
	for name, w := range want {
		switch w := w.(type) {
		case *big.Int:
			if have, ok := fields[name].(*big.Int); !ok || have.Cmp(w) != 0 {
				t.Errorf("event field %s: have %v, want %v", name, fields[name], w)
			}
		case []byte:
			if have, ok := fields[name].([]byte); !ok || !bytes.Equal(have, w) {
				t.Errorf("event field %s: have %x, want %x", name, fields[name], w)
			}
		default:
			if fields[name] != w {
				t.Errorf("event field %s: have %v, want %v", name, fields[name], w)
			}
		}
	}
}

func TestXShardOutboxInvalid(t *testing.T) {
	var (
		cost  = uint64(xShardValue + xShardGasLimit*xShardGasPrice)
		input = func(t *testing.T) []byte {
			return packXShardSend(t, xShardDst, xShardRecipient, big.NewInt(xShardValue), xShardGasLimit, []byte{1, 2, 3})
		}
		// edit returns the default input with args[off] set to b.
		edit = func(off int, b byte) func(t *testing.T) []byte {
			return func(t *testing.T) []byte {
				in := input(t)
				in[4+off] = b
				return in
			}
		}
		maxUint256 = new(big.Int).Sub(new(big.Int).Lsh(common.Big1, 256), common.Big1)
	)
	for _, tt := range []struct {
		name   string
		input  func(t *testing.T) []byte
		value  uint64
		caller common.Address // defaults to xShardSender
		call   string         // defaults to CALL
		static bool           // run the CALL in a static context
		want   error
	}{
		{name: "empty input", input: func(*testing.T) []byte { return nil }, want: errXShardInvalidInput},
		{name: "unknown selector", input: func(t *testing.T) []byte { in := input(t); in[0] ^= 1; return in }, want: errXShardInvalidInput},
		{name: "short input", input: func(t *testing.T) []byte { return input(t)[:4+5*32] }, want: errXShardInvalidInput},
		{name: "dirty dstShard", input: edit(27, 1), want: errXShardInvalidInput},
		{name: "dirty to", input: edit(32+11, 1), want: errXShardInvalidInput},
		{name: "dirty gasLimit", input: edit(3*32+23, 1), want: errXShardInvalidInput},
		{name: "non-canonical offset", input: edit(4*32+31, 0xc0), want: errXShardInvalidInput},
		{name: "length past input", input: edit(5*32+31, 33), want: errXShardInvalidInput},
		{name: "huge length", input: func(t *testing.T) []byte {
			in := packXShardSend(t, xShardDst, xShardRecipient, big.NewInt(xShardValue), xShardGasLimit, nil)
			copy(in[4+5*32+24:], bytes.Repeat([]byte{0xff}, 8))
			return in
		}, want: errXShardInvalidInput},
		{name: "trailing bytes", input: func(t *testing.T) []byte { return append(input(t), make([]byte, 32)...) }, want: errXShardInvalidInput},
		{name: "missing padding", input: func(t *testing.T) []byte { in := input(t); return in[:len(in)-1] }, want: errXShardInvalidInput},
		{name: "own shard", input: func(t *testing.T) []byte {
			return packXShardSend(t, 1, xShardRecipient, big.NewInt(xShardValue), xShardGasLimit, nil)
		}, want: errXShardInvalidShard},
		{name: "unknown shard", input: func(t *testing.T) []byte {
			return packXShardSend(t, 3, xShardRecipient, big.NewInt(xShardValue), xShardGasLimit, nil)
		}, want: errXShardInvalidShard},
		{name: "gas limit below minimum", input: func(t *testing.T) []byte {
			return packXShardSend(t, xShardDst, xShardRecipient, big.NewInt(xShardValue), params.XShardSendGas-1, nil)
		}, want: errXShardInvalidGasLimit},
		{name: "gas limit above max", input: func(t *testing.T) []byte {
			return packXShardSend(t, xShardDst, xShardRecipient, big.NewInt(xShardValue), params.MaxTxGas+1, nil)
		}, want: errXShardInvalidGasLimit},
		{name: "insufficient value", input: input, value: cost - 1, want: errXShardInsufficientValue},
		{name: "cost overflow", input: func(t *testing.T) []byte {
			return packXShardSend(t, xShardDst, xShardRecipient, maxUint256, xShardGasLimit, nil)
		}, want: errXShardInsufficientValue},
		{name: "contract sender", input: input, caller: common.HexToAddress("0xc0de"), want: errXShardSenderNotOrigin},
		{name: "system call", input: input, caller: params.SystemAddress, want: errXShardCallOnly},
		{name: "static context", input: input, static: true, want: ErrWriteProtection},
		{name: "staticcall", input: input, call: "STATICCALL", want: errXShardCallOnly},
		{name: "delegatecall", input: input, call: "DELEGATECALL", want: errXShardCallOnly},
		{name: "callcode", input: input, call: "CALLCODE", want: errXShardCallOnly},
	} {
		t.Run(tt.name, func(t *testing.T) {
			evm, statedb := newXShardEVM()
			caller := xShardSender
			if tt.caller != (common.Address{}) {
				caller = tt.caller
				statedb.AddBalance(caller, xShardFunds, tracing.BalanceChangeUnspecified)
			}
			value := uint256.NewInt(cost)
			if tt.value != 0 || tt.static {
				value.SetUint64(tt.value)
			}
			var (
				in  = tt.input(t)
				gas = NewGasBudget(1_000_000, 0)
				err error
			)
			switch tt.call {
			case "":
				evm.readOnly = tt.static
				_, _, err = evm.Call(caller, params.XShardOutboxAddress, in, gas, value)
			case "STATICCALL":
				_, _, err = evm.StaticCall(caller, params.XShardOutboxAddress, in, gas)
			case "DELEGATECALL":
				_, _, err = evm.DelegateCall(caller, caller, params.XShardOutboxAddress, in, gas, value)
			case "CALLCODE":
				_, _, err = evm.CallCode(caller, params.XShardOutboxAddress, in, gas, value)
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("error %v, want %v", err, tt.want)
			}
			if have := statedb.GetBalance(caller); !have.Eq(xShardFunds) {
				t.Errorf("caller balance %v, want %v", have, xShardFunds)
			}
			if logs := statedb.Logs(); len(logs) != 0 {
				t.Errorf("got %d logs, want none", len(logs))
			}
		})
	}
}

func TestXShardOutboxActivation(t *testing.T) {
	var (
		ethRules = params.MergedTestChainConfig.Rules(common.Big0, true, 0)
		qkcRules = xShardChainConfig().Rules(common.Big0, true, 0)
		ethAddrs = ActivePrecompiles(ethRules)
	)
	if !qkcRules.IsQuarkChain || ethRules.IsQuarkChain {
		t.Fatalf("IsQuarkChain: have %v on QuarkChain and %v on Ethereum", qkcRules.IsQuarkChain, ethRules.IsQuarkChain)
	}
	if slices.Contains(ethAddrs, params.XShardOutboxAddress) {
		t.Error("outbox is active on Ethereum")
	}
	if _, ok := ActivePrecompiledContracts(ethRules)[params.XShardOutboxAddress]; ok {
		t.Error("outbox is in the Ethereum precompiled contracts")
	}
	// The QuarkChain precompiles extend the Ethereum ones of the same fork.
	qkcAddrs := ActivePrecompiles(qkcRules)
	if want := append(slices.Clone(ethAddrs), params.XShardOutboxAddress); !slices.Equal(qkcAddrs, want) {
		t.Errorf("QuarkChain precompiles %v, want %v", qkcAddrs, want)
	}
	if !slices.Equal(ActivePrecompiles(ethRules), ethAddrs) {
		t.Error("Ethereum precompiles changed")
	}
	if _, ok := ActivePrecompiledContracts(qkcRules)[params.XShardOutboxAddress]; !ok {
		t.Error("outbox is not in the QuarkChain precompiled contracts")
	}
	if _, ok := (*activePrecompiledContracts(qkcRules))[params.XShardOutboxAddress]; ok {
		t.Error("outbox leaked into the shared Ethereum precompile set")
	}
	evm, _ := newXShardEVM()
	if _, ok := evm.precompile(params.XShardOutboxAddress); !ok {
		t.Error("outbox is not a precompile of the QuarkChain EVM")
	}
}
