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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
)

// TraceTypes selects the independent output families of the trace namespace.
type TraceTypes []string

func (t *TraceTypes) UnmarshalJSON(input []byte) error {
	if bytes.Equal(bytes.TrimSpace(input), []byte("null")) {
		return fmt.Errorf("trace types must be an array")
	}
	type plain TraceTypes
	if err := json.Unmarshal(input, (*plain)(t)); err != nil {
		return err
	}
	return t.validate()
}

func (t TraceTypes) validate() error {
	seen := make(map[string]bool)
	for _, name := range t {
		if name != "trace" && name != "stateDiff" && name != "vmTrace" {
			return traceInvalid("unknown trace type %q", name)
		}
		if seen[name] {
			return traceInvalid("duplicate trace type %q", name)
		}
		seen[name] = true
	}
	return nil
}

func (t TraceTypes) has(name string) bool {
	for _, s := range t {
		if s == name {
			return true
		}
	}
	return false
}

// TraceCallArgs validates known unsigned-call fields and ignores extensions.
type TraceCallArgs struct {
	ethapi.TransactionArgs
	Type *hexutil.Uint64 `json:"type,omitempty"`
}

func (a *TraceCallArgs) UnmarshalJSON(input []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(input, &fields); err != nil {
		return err
	}
	if fields == nil {
		return fmt.Errorf("call must be an object")
	}
	*a = TraceCallArgs{}
	for key, value := range fields {
		switch key {
		case "from", "to", "gas", "gasPrice", "maxFeePerGas", "maxPriorityFeePerGas", "value", "data", "input", "nonce", "type", "accessList", "chainId", "maxFeePerBlobGas", "blobVersionedHashes", "authorizationList":
		default:
			delete(fields, key)
			continue
		}
		if key != "to" && bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("%s must not be null", key)
		}
	}
	if raw, ok := fields["type"]; ok {
		var encoded string
		if err := json.Unmarshal(raw, &encoded); err != nil {
			return err
		}
		// The profile uses a hex byte, allowing both 0x2 and 0x02.
		if len(encoded) == 3 && encoded[:2] == "0x" {
			encoded = "0x0" + encoded[2:]
		}
		decoded, err := hexutil.Decode(encoded)
		if err != nil || len(decoded) != 1 {
			return fmt.Errorf("type must be a hex-encoded byte")
		}
		kind := hexutil.Uint64(decoded[0])
		if kind > types.SetCodeTxType {
			return fmt.Errorf("unsigned trace calls support transaction types 0 through 4")
		}
		a.Type = &kind
	}
	// Filter before decoding so fields outside the profile cannot accidentally
	// acquire meaning from TransactionArgs (for example, blob sidecars).
	filtered, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	return json.Unmarshal(filtered, &a.TransactionArgs)
}

// TraceCallManyEntry is exactly one [call, traceTypes] pair.
type TraceCallManyEntry struct {
	Call  TraceCallArgs
	Types TraceTypes
}

func (e *TraceCallManyEntry) UnmarshalJSON(input []byte) error {
	var pair []json.RawMessage
	if err := json.Unmarshal(input, &pair); err != nil {
		return err
	}
	if len(pair) != 2 {
		return fmt.Errorf("expected [call, traceTypes]")
	}
	if err := json.Unmarshal(pair[0], &e.Call); err != nil {
		return err
	}
	return json.Unmarshal(pair[1], &e.Types)
}

// TraceCalls is a non-null list of sequential call pairs.
type TraceCalls []TraceCallManyEntry

func (c *TraceCalls) UnmarshalJSON(input []byte) error {
	if bytes.Equal(bytes.TrimSpace(input), []byte("null")) {
		return fmt.Errorf("calls must be an array")
	}
	type plain TraceCalls
	return json.Unmarshal(input, (*plain)(c))
}

// TracePosition identifies a path in a call tree, not an index in a flat list.
type TracePosition []hexutil.Uint64

func (p *TracePosition) UnmarshalJSON(input []byte) error {
	if bytes.Equal(bytes.TrimSpace(input), []byte("null")) {
		return fmt.Errorf("position must be an array")
	}
	type plain TracePosition
	return json.Unmarshal(input, (*plain)(p))
}

// TraceFilter describes an inclusive block range and pagination after matching.
type TraceFilter struct {
	FromBlock   *rpc.BlockNumber `json:"fromBlock"`
	ToBlock     *rpc.BlockNumber `json:"toBlock"`
	FromAddress []common.Address `json:"fromAddress"`
	ToAddress   []common.Address `json:"toAddress"`
	Mode        TraceFilterMode  `json:"mode"`
	After       *uint64          `json:"after"`
	Count       *uint64          `json:"count"`
}

// TraceFilterMode selects how populated address lists combine. The zero value is intersection.
type TraceFilterMode string

const (
	TraceFilterIntersection TraceFilterMode = "intersection"
	TraceFilterUnion        TraceFilterMode = "union"
)

func (m *TraceFilterMode) UnmarshalJSON(input []byte) error {
	var value string
	if err := json.Unmarshal(input, &value); err != nil {
		return fmt.Errorf("mode must be a string")
	}
	switch mode := TraceFilterMode(value); mode {
	case TraceFilterIntersection, TraceFilterUnion:
		*m = mode
		return nil
	}
	return fmt.Errorf("unknown filter mode %q", value)
}

func (f *TraceFilter) UnmarshalJSON(input []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(input, &fields); err != nil {
		return err
	}
	if fields == nil {
		return fmt.Errorf("filter must be an object")
	}
	// Null address lists are unrestricted, like omitted or empty lists.
	for key, value := range fields {
		if key != "fromAddress" && key != "toAddress" && bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("%s must not be null", key)
		}
	}
	type plain TraceFilter
	dec := json.NewDecoder(bytes.NewReader(input))
	dec.DisallowUnknownFields()
	return dec.Decode((*plain)(f))
}

// TraceExecution contains actual output even when no tracer output is selected.
type TraceExecution struct {
	Output          hexutil.Bytes                        `json:"output"`
	Trace           []*TraceFrame                        `json:"trace"`
	StateDiff       map[common.Address]*traceAccountDiff `json:"stateDiff"`
	VMTrace         *traceVM                             `json:"vmTrace"`
	TransactionHash *common.Hash                         `json:"transactionHash,omitempty"`
}

// TraceFrame is one preorder call-tree record.
type TraceFrame struct {
	Action       any      `json:"action"`
	Result       any      `json:"result"`
	Error        string   `json:"error,omitempty"`
	Subtraces    uint64   `json:"subtraces"`
	TraceAddress []uint64 `json:"traceAddress"`
	Type         string   `json:"type"`
	*traceLocation
}

type traceLocation struct {
	BlockHash           common.Hash  `json:"blockHash"`
	BlockNumber         uint64       `json:"blockNumber"`
	TransactionHash     *common.Hash `json:"transactionHash"`
	TransactionPosition *uint64      `json:"transactionPosition"`
}

type traceCallAction struct {
	CallType string         `json:"callType"`
	From     common.Address `json:"from"`
	To       common.Address `json:"to"`
	Gas      hexutil.Uint64 `json:"gas"`
	Input    hexutil.Bytes  `json:"input"`
	Value    *hexutil.Big   `json:"value"`
}

type traceCreateAction struct {
	From           common.Address `json:"from"`
	Gas            hexutil.Uint64 `json:"gas"`
	Init           hexutil.Bytes  `json:"init"`
	Value          *hexutil.Big   `json:"value"`
	CreationMethod string         `json:"creationMethod"`
}

type traceSuicideAction struct {
	Address       common.Address `json:"address"`
	RefundAddress common.Address `json:"refundAddress"`
	Balance       *hexutil.Big   `json:"balance"`
}

type traceRewardAction struct {
	Author     common.Address `json:"author"`
	Value      *hexutil.Big   `json:"value"`
	RewardType string         `json:"rewardType"`
}

type traceCallResult struct {
	GasUsed hexutil.Uint64 `json:"gasUsed"`
	Output  hexutil.Bytes  `json:"output"`
}

type traceCreateResult struct {
	GasUsed hexutil.Uint64 `json:"gasUsed"`
	Address common.Address `json:"address"`
	Code    hexutil.Bytes  `json:"code"`
}

type traceAccountDiff struct {
	Balance any                 `json:"balance"`
	Nonce   any                 `json:"nonce"`
	Code    any                 `json:"code"`
	Storage map[common.Hash]any `json:"storage"`
}

type traceVM struct {
	Code hexutil.Bytes `json:"code"`
	Ops  []*traceVMOp  `json:"ops"`
}

type traceVMOp struct {
	PC   uint64          `json:"pc"`
	Cost uint64          `json:"cost"`
	Ex   *traceVMEffects `json:"ex"`
	Sub  *traceVM        `json:"sub"`
	Op   string          `json:"op"`
}

type traceVMEffects struct {
	Used  uint64             `json:"used"`
	Push  []*hexutil.Big     `json:"push"`
	Mem   *traceMemoryWrite  `json:"mem"`
	Store *traceStorageWrite `json:"store"`
}

type traceMemoryWrite struct {
	Off  uint64        `json:"off"`
	Data hexutil.Bytes `json:"data"`
}
type traceStorageWrite struct {
	Key *hexutil.Big `json:"key"`
	Val *hexutil.Big `json:"val"`
}

type traceRPCError struct {
	code    int
	message string
}

func (e *traceRPCError) Error() string  { return e.message }
func (e *traceRPCError) ErrorCode() int { return e.code }
func traceInvalid(format string, args ...any) error {
	return &traceRPCError{-32602, fmt.Sprintf(format, args...)}
}

// traceItemError reports the zero-based index of the failing trace_callMany
// item in error.data.
type traceItemError struct {
	err   rpc.Error
	index int
}

func (e *traceItemError) Error() string  { return fmt.Sprintf("call %d: %v", e.index, e.err) }
func (e *traceItemError) ErrorCode() int { return e.err.ErrorCode() }
func (e *traceItemError) ErrorData() any { return map[string]int{"index": e.index} }

// traceCallRejection maps a rejected unsigned call to the eth_simulateV1
// validation codes. Nonce and sender-code checks are skipped for unsigned
// calls, as in eth_call; other rejections are invalid parameters.
func traceCallRejection(err error) error {
	code := -32602
	switch {
	case errors.Is(err, core.ErrFeeCapTooLow):
		code = -38012
	case errors.Is(err, core.ErrIntrinsicGas), errors.Is(err, core.ErrFloorDataGas):
		code = -38013
	case errors.Is(err, core.ErrInsufficientFunds), errors.Is(err, core.ErrInsufficientFundsForTransfer):
		code = -38014
	case errors.Is(err, vm.ErrMaxInitCodeSizeExceeded):
		code = -38025
	}
	return &traceRPCError{code, err.Error()}
}

// traceRawRejection maps a rejected signed transaction to the
// eth_sendRawTransaction error groups, with -32003 for other rejections.
func traceRawRejection(err error) error {
	code := -32003
	switch {
	case errors.Is(err, core.ErrNonceTooLow):
		code = 1
	case errors.Is(err, core.ErrNonceTooHigh):
		code = 2
	case errors.Is(err, core.ErrIntrinsicGas), errors.Is(err, core.ErrFloorDataGas):
		code = 800
	case errors.Is(err, core.ErrTipAboveFeeCap):
		code = 804
	case errors.Is(err, core.ErrFeeCapTooLow):
		code = 806
	case errors.Is(err, core.ErrInsufficientFunds), errors.Is(err, core.ErrInsufficientFundsForTransfer):
		code = 809
	}
	return &traceRPCError{code, err.Error()}
}

// traceRejected reports a mined transaction that failed validation on replay.
func traceRejected(err error) error { return &traceRPCError{-32003, err.Error()} }
