// Copyright 2022 The go-ethereum Authors
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

package native

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/eth/tracers"
)

//go:generate go run github.com/fjl/gencodec -type flatCallAction -field-override flatCallActionMarshaling -out gen_flatcallaction_json.go
//go:generate go run github.com/fjl/gencodec -type flatCallResult -field-override flatCallResultMarshaling -out gen_flatcallresult_json.go

func init() {
	// tracers.DefaultDirectory.Register("flatCallTracer", newFlatCallTracer, false)
	register("flatCallTracer", newFlatCallTracer)
}

var parityErrorMapping = map[string]string{
	"contract creation code storage out of gas": "Out of gas",
	"out of gas":                      "Out of gas",
	"gas uint64 overflow":             "Out of gas",
	"max code size exceeded":          "Out of gas",
	"invalid jump destination":        "Bad jump destination",
	"execution reverted":              "Reverted",
	"return data out of bounds":       "Out of bounds",
	"stack limit reached 1024 (1023)": "Out of stack",
	"precompiled failed":              "Built-in failed",
	"invalid input length":            "Built-in failed",
}

var parityErrorMappingStartingWith = map[string]string{
	"invalid opcode:": "Bad instruction",
	"stack underflow": "Stack underflow",
}

// flatCallFrame is a standalone callframe.
type flatCallFrame struct {
	Action              flatCallAction  `json:"action"`
	BlockHash           *common.Hash    `json:"blockHash"`
	BlockNumber         uint64          `json:"blockNumber"`
	Error               string          `json:"error,omitempty"`
	Result              *flatCallResult `json:"result,omitempty"`
	Subtraces           int             `json:"subtraces"`
	TraceAddress        []int           `json:"traceAddress"`
	TransactionHash     *common.Hash    `json:"transactionHash"`
	TransactionPosition uint64          `json:"transactionPosition"`
	Type                string          `json:"type"`
}

type flatCallAction struct {
	Author         *common.Address `json:"author,omitempty"`
	RewardType     string          `json:"rewardType,omitempty"`
	SelfDestructed *common.Address `json:"address,omitempty"`
	Balance        *big.Int        `json:"balance,omitempty"`
	CallType       string          `json:"callType,omitempty"`
	CreationMethod string          `json:"creationMethod,omitempty"`
	From           *common.Address `json:"from,omitempty"`
	Gas            *uint64         `json:"gas,omitempty"`
	Init           *[]byte         `json:"init,omitempty"`
	Input          *[]byte         `json:"input,omitempty"`
	RefundAddress  *common.Address `json:"refundAddress,omitempty"`
	To             *common.Address `json:"to,omitempty"`
	Value          *big.Int        `json:"value,omitempty"`
}

type flatCallActionMarshaling struct {
	Balance *hexutil.Big
	Gas     *hexutil.Uint64
	Init    *hexutil.Bytes
	Input   *hexutil.Bytes
	Value   *hexutil.Big
}

type flatCallResult struct {
	Address *common.Address `json:"address,omitempty"`
	Code    *[]byte         `json:"code,omitempty"`
	GasUsed *uint64         `json:"gasUsed,omitempty"`
	Output  *[]byte         `json:"output,omitempty"`
}

type flatCallResultMarshaling struct {
	Code    *hexutil.Bytes
	GasUsed *hexutil.Uint64
	Output  *hexutil.Bytes
}

// flatCallTracer reports call frame information of a tx in a flat format, i.e.
// as opposed to the nested format of `callTracer`.
type flatCallTracer struct {
	ctx *tracers.Context
	*callTracer
}

// newFlatCallTracer returns a new flatCallTracer.
func newFlatCallTracer(ctx *tracers.Context) tracers.Tracer {
	t := &callTracer{callstack: make([]callFrame, 1)}

	return &flatCallTracer{callTracer: t, ctx: ctx}

}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *flatCallTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	t.callTracer.CaptureEnter(typ, from, to, input, gas, value)
	if len(t.callTracer.callstack) > 0 && t.callTracer.callstack[len(t.callTracer.callstack)-1].Value == "" {
		t.callTracer.callstack[len(t.callTracer.callstack)-1].Value = "0x0"
	}
}

// GetResult returns the json-encoded nested list of call traces, and any
// error arising from the encoding or forceful termination (via `Stop`).
func (t *flatCallTracer) GetResult() (json.RawMessage, error) {
	if len(t.callTracer.callstack) < 1 {
		return nil, errors.New("invalid number of calls")
	}

	flat, err := flatFromNested(&t.callTracer.callstack[0], []int{}, true, t.ctx)
	if err != nil {
		return nil, err
	}

	res, err := json.Marshal(flat)
	if err != nil {
		return nil, err
	}
	return res, t.reason

}

func flatFromNested(input *callFrame, traceAddress []int, convertErrs bool, ctx *tracers.Context) (output []flatCallFrame, err error) {
	var frame *flatCallFrame
	switch vm.StringToOp(input.Type) {
	case vm.CREATE, vm.CREATE2:
		frame = newFlatCreate(input)
	case vm.SELFDESTRUCT:
		frame = newFlatSuicide(input)
	case vm.CALL, vm.STATICCALL, vm.CALLCODE, vm.DELEGATECALL:
		frame = newFlatCall(input)
	default:
		return nil, fmt.Errorf("unrecognized call frame type: %s", input.Type)
	}

	frame.TraceAddress = traceAddress
	frame.Error = input.Error
	frame.Subtraces = len(input.Calls)
	fillCallFrameFromContext(frame, ctx)
	if convertErrs {
		convertErrorToParity(frame)
	}

	// Revert output contains useful information (revert reason).
	// Otherwise discard result.
	if input.Error != "" && input.Error != vm.ErrExecutionReverted.Error() {
		frame.Result = nil
	}

	output = append(output, *frame)
	if len(input.Calls) > 0 {
		for i, childCall := range input.Calls {
			childAddr := childTraceAddress(traceAddress, i)
			childCallCopy := childCall
			flat, err := flatFromNested(&childCallCopy, childAddr, convertErrs, ctx)
			if err != nil {
				return nil, err
			}
			output = append(output, flat...)
		}
	}

	return output, nil
}

func addressPointer(a string) *common.Address {
	if a == "" {
		return nil
	}
	addr := common.HexToAddress(a)
	return &addr
}

func uint64Pointer(v string) *uint64 {
	if v == "" {
		return nil
	}
	val, _ := hexutil.DecodeUint64(v)
	return &val
}

func bigInt(v string) *big.Int {
	if v == "" {
		return nil
	}
	val, _ := hexutil.DecodeBig(v)
	return val
}

func bytesPointer(v string) *[]byte {
	if v == "" {
		return nil
	}
	val := hexutil.MustDecode(v)
	return &val

}

func newFlatCreate(input *callFrame) *flatCallFrame {
	var (
		actionInit = bytesPointer(input.Input)
		resultCode = bytesPointer(input.Output)
	)

	return &flatCallFrame{
		Type: strings.ToLower(vm.CREATE.String()),
		Action: flatCallAction{
			From:  addressPointer(input.From),
			Gas:   uint64Pointer(input.Gas),
			Value: bigInt(input.Value),
			Init:  actionInit,
		},
		Result: &flatCallResult{
			GasUsed: uint64Pointer(input.GasUsed),
			Address: addressPointer(input.To),
			Code:    resultCode,
		},
	}
}

func newFlatCall(input *callFrame) *flatCallFrame {
	var (
		actionInput  = bytesPointer(input.Input)
		resultOutput = bytesPointer(input.Output)
	)

	return &flatCallFrame{
		Type: strings.ToLower(vm.CALL.String()),
		Action: flatCallAction{
			From:     addressPointer(input.From),
			To:       addressPointer(input.To),
			Gas:      uint64Pointer(input.Gas),
			Value:    bigInt(input.Value),
			CallType: strings.ToLower(input.Type),
			Input:    actionInput,
		},
		Result: &flatCallResult{
			GasUsed: uint64Pointer(input.GasUsed),
			Output:  resultOutput,
		},
	}
}

func newFlatSuicide(input *callFrame) *flatCallFrame {
	return &flatCallFrame{
		Type: "suicide",
		Action: flatCallAction{
			SelfDestructed: addressPointer(input.From),
			Balance:        bigInt(input.Value),
			RefundAddress:  addressPointer(input.To),
		},
	}
}

func fillCallFrameFromContext(callFrame *flatCallFrame, ctx *tracers.Context) {
	if ctx.BlockHash != (common.Hash{}) {
		callFrame.BlockHash = &ctx.BlockHash
	}
	if ctx.BlockNumber != 0 {
		callFrame.BlockNumber = ctx.BlockNumber
	}
	if ctx.TxHash != (common.Hash{}) {
		callFrame.TransactionHash = &ctx.TxHash
	}
	callFrame.TransactionPosition = uint64(ctx.TxIndex)
}

func convertErrorToParity(call *flatCallFrame) {
	if call.Error == "" {
		return
	}

	if parityError, ok := parityErrorMapping[call.Error]; ok {
		call.Error = parityError
	} else {
		for gethError, parityError := range parityErrorMappingStartingWith {
			if strings.HasPrefix(call.Error, gethError) {
				call.Error = parityError
			}
		}
	}
}

func childTraceAddress(a []int, i int) []int {
	child := make([]int, 0, len(a)+1)
	child = append(child, a...)
	child = append(child, i)
	return child
}
