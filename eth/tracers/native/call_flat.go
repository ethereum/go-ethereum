// Copyright 2021 The go-ethereum Authors
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
	"math/big"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

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

func init() {
	register("flatCallTracer", newFlatCallTracer)
}

// callParityFrame is the result of a callParityTracerParity run.
type callParityFrame struct {
	Action              CallTraceParityAction  `json:"action"`
	BlockHash           *common.Hash           `json:"blockHash"`
	BlockNumber         uint64                 `json:"blockNumber"`
	Error               string                 `json:"error,omitempty"`
	Result              *CallTraceParityResult `json:"result,omitempty"`
	Subtraces           int                    `json:"subtraces"`
	TraceAddress        []int                  `json:"traceAddress"`
	TransactionHash     *common.Hash           `json:"transactionHash"`
	TransactionPosition *uint64                `json:"transactionPosition"`
	Type                string                 `json:"type"`
	Calls               []callParityFrame      `json:"-"`
}

type CallTraceParityAction struct {
	Author         *common.Address `json:"author,omitempty"`
	RewardType     *string         `json:"rewardType,omitempty"`
	SelfDestructed *common.Address `json:"address,omitempty"`
	Balance        *hexutil.Big    `json:"balance,omitempty"`
	CallType       string          `json:"callType,omitempty"`
	CreationMethod string          `json:"creationMethod,omitempty"`
	From           *common.Address `json:"from,omitempty"`
	Gas            *hexutil.Uint64 `json:"gas,omitempty"`
	Init           *hexutil.Bytes  `json:"init,omitempty"`
	Input          *hexutil.Bytes  `json:"input,omitempty"`
	RefundAddress  *common.Address `json:"refundAddress,omitempty"`
	To             *common.Address `json:"to,omitempty"`
	Value          *hexutil.Big    `json:"value,omitempty"`
}

type CallTraceParityResult struct {
	Address *common.Address `json:"address,omitempty"`
	Code    *hexutil.Bytes  `json:"code,omitempty"`
	GasUsed *hexutil.Uint64 `json:"gasUsed,omitempty"`
	Output  *hexutil.Bytes  `json:"output,omitempty"`
}
type callParityTracer struct {
	env               *vm.EVM
	config            callTracerConfig
	ctx               *tracers.Context // Holds tracer context data
	callstack         []callParityFrame
	interrupt         uint32           // Atomic flag to signal execution interruption
	reason            error            // Textual reason for the interruption
	activePrecompiles []common.Address // Updated on CaptureStart based on given rules
}

func (t *callParityTracer) CaptureTxStart(gasLimit uint64) {}

func (t *callParityTracer) CaptureTxEnd(restGas uint64) {}

// NewCallParityTracer returns a native go tracer which tracks
// call frames of a tx, and implements vm.EVMLogger.
func newFlatCallTracer(ctx *tracers.Context, cfg json.RawMessage) (tracers.Tracer, error) {
	var config callTracerConfig
	if cfg != nil {
		if err := json.Unmarshal(cfg, &config); err != nil {
			return nil, err
		}
	}
	// First callframe contains tx context info
	// and is populated on start and end.
	return &callParityTracer{callstack: make([]callParityFrame, 1), config: config}, nil
}

// isPrecompiled returns whether the addr is a precompile. Logic borrowed from newJsTracer in eth/tracers/js/tracer.go
func (t *callParityTracer) isPrecompiled(addr common.Address) bool {
	for _, p := range t.activePrecompiles {
		if p == addr {
			return true
		}
	}
	return false
}

func (t *callParityTracer) fillCallFrameFromContext(callFrame *callParityFrame) {
	if t.ctx != nil {
		if t.ctx.BlockHash != (common.Hash{}) {
			callFrame.BlockHash = &t.ctx.BlockHash
		}
		if t.ctx.TxHash != (common.Hash{}) {
			callFrame.TransactionHash = &t.ctx.TxHash
		}
		transactionPosition := uint64(t.ctx.TxIndex)
		callFrame.TransactionPosition = &transactionPosition
	}
}

func (t *callParityTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.env = env

	// Skip any pre-compile invocations, those are just fancy opcodes
	rules := env.ChainConfig().Rules(env.Context.BlockNumber, env.Context.Random != nil)
	t.activePrecompiles = vm.ActivePrecompiles(rules)

	inputHex := hexutil.Bytes(common.CopyBytes(input))
	gasHex := hexutil.Uint64(gas)

	t.callstack[0] = callParityFrame{
		Type: strings.ToLower(vm.CALL.String()),
		Action: CallTraceParityAction{
			From:  &from,
			To:    &to,
			Input: &inputHex,
			Gas:   &gasHex,
		},
		Result:      &CallTraceParityResult{},
		BlockNumber: env.Context.BlockNumber.Uint64(),
	}
	if value != nil {
		valueHex := hexutil.Big(*value)
		t.callstack[0].Action.Value = &valueHex
	}
	if create {
		t.callstack[0].Type = strings.ToLower(vm.CREATE.String())
	}

	t.fillCallFrameFromContext(&t.callstack[0])
}

func (t *callParityTracer) CaptureEnd(output []byte, gasUsed uint64, _ time.Duration, err error) {
	if err != nil {
		t.callstack[0].Error = err.Error()
		if err.Error() == "execution reverted" && len(output) > 0 {
			outputHex := hexutil.Bytes(common.CopyBytes(output))
			t.callstack[0].Result.Output = &outputHex
		}
	} else {
		// TODO (ziogaschr): move back outside of if, makes sense to have it always. Is addition, no API breaks
		gasUsedHex := hexutil.Uint64(gasUsed)
		t.callstack[0].Result.GasUsed = &gasUsedHex

		outputHex := hexutil.Bytes(common.CopyBytes(output))
		t.callstack[0].Result.Output = &outputHex
	}
}

func (t *callParityTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
}

func (t *callParityTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
}

func (t *callParityTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// Skip if tracing was interrupted
	if atomic.LoadUint32(&t.interrupt) > 0 {
		t.env.Cancel()
		return
	}

	inputHex := hexutil.Bytes(common.CopyBytes(input))
	gasHex := hexutil.Uint64(gas)

	call := callParityFrame{
		Type: strings.ToLower(typ.String()),
		Action: CallTraceParityAction{
			From:  &from,
			To:    &to,
			Input: &inputHex,
			Gas:   &gasHex,
		},
		Result:      &CallTraceParityResult{},
		BlockNumber: t.callstack[0].BlockNumber,
	}
	valueHex := hexutil.Big{}
	if value != nil {
		valueHex = hexutil.Big(*value)
	}
	call.Action.Value = &valueHex

	t.fillCallFrameFromContext(&call)

	t.callstack = append(t.callstack, call)
}

func (t *callParityTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	size := len(t.callstack)
	if size <= 1 {
		return
	}
	// pop call
	call := t.callstack[size-1]
	t.callstack = t.callstack[:size-1]
	size -= 1

	// Skip any pre-compile invocations, those are just fancy opcodes
	// NOTE: let them captured on `CaptureEnter` method so as we handle internal txs state correctly
	//			 and drop them here, as it has been removed from the callstack
	typ := vm.StringToOp(strings.ToUpper(call.Type))
	if t.isPrecompiled(*call.Action.To) && (typ == vm.CALL || typ == vm.STATICCALL) {
		return
	}

	gasUsedHex := hexutil.Uint64(gasUsed)
	call.Result.GasUsed = &gasUsedHex
	if err == nil {
		outputHex := hexutil.Bytes(common.CopyBytes(output))
		call.Result.Output = &outputHex
	} else {
		call.Error = err.Error()
		typ := vm.StringToOp(strings.ToUpper(call.Type))
		if typ == vm.CREATE || typ == vm.CREATE2 {
			call.Action.To = nil
		}
	}
	t.callstack[size-1].Calls = append(t.callstack[size-1].Calls, call)
}

func (t *callParityTracer) Finalize(call callParityFrame, traceAddress []int) ([]callParityFrame, error) {
	typ := vm.StringToOp(strings.ToUpper(call.Type))
	if typ == vm.CREATE || typ == vm.CREATE2 {
		t.formatCreateResult(&call)
	} else if typ == vm.SELFDESTRUCT {
		t.formatSuicideResult(&call)
	} else {
		t.formatCallResult(&call)
	}

	// for _, errorContains := range paritySkipTracesForErrors {
	// 	if strings.Contains(call.Error, errorContains) {
	// 		return
	// 	}
	// }

	t.convertErrorToParity(&call)

	if subtraces := len(call.Calls); subtraces > 0 {
		call.Subtraces = subtraces
	}

	call.TraceAddress = traceAddress

	results := []callParityFrame{call}

	for i := 0; i < len(call.Calls); i++ {
		childCall := call.Calls[i]

		var childTraceAddress []int
		childTraceAddress = append(childTraceAddress, traceAddress...)
		childTraceAddress = append(childTraceAddress, i)

		// Delegatecall uses the value from parent, if zero
		childCallType := vm.StringToOp(strings.ToUpper(childCall.Type))
		if (childCallType == vm.DELEGATECALL) &&
			(childCall.Action.Value == nil || childCall.Action.Value.ToInt().Cmp(big.NewInt(0)) == 0) {
			childCall.Action.Value = call.Action.Value
		}

		child, err := t.Finalize(childCall, childTraceAddress)
		if err != nil {
			return nil, errors.New("failed to parse trace frame")
		}

		results = append(results, child...)
	}

	return results, nil
}

func (t *callParityTracer) GetResult() (json.RawMessage, error) {
	if len(t.callstack) != 1 {
		return nil, errors.New("incorrect number of top-level calls")
	}

	traceAddress := []int{}
	result, err := t.Finalize(t.callstack[0], traceAddress)
	if err != nil {
		return nil, err
	}

	res, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(res), t.reason
}

func (t *callParityTracer) Stop(err error) {
	t.reason = err
	atomic.StoreUint32(&t.interrupt, 1)
}

func (t *callParityTracer) formatCreateResult(call *callParityFrame) {
	call.Type = strings.ToLower(vm.CREATE.String())

	input := call.Action.Input
	call.Action.Init = input

	to := call.Action.To
	call.Result.Address = to

	call.Result.Code = call.Result.Output

	call.Action.To = nil
	call.Action.Input = nil

	call.Result.Output = nil
}

func (t *callParityTracer) formatCallResult(call *callParityFrame) {
	call.Action.CallType = call.Type

	typ := vm.StringToOp(strings.ToUpper(call.Type))

	// update after callResult so as it affects only the root type
	if typ == vm.CALLCODE || typ == vm.DELEGATECALL || typ == vm.STATICCALL {
		call.Type = strings.ToLower(vm.CALL.String())
	}
}

func (t *callParityTracer) formatSuicideResult(call *callParityFrame) {
	call.Type = "suicide"

	addrFrom := call.Action.From
	call.Action.SelfDestructed = addrFrom

	balanceHex := *call.Action.Value
	call.Action.Balance = &balanceHex

	call.Action.Value = nil

	addrTo := call.Action.To
	call.Action.RefundAddress = addrTo

	call.Action.From = nil
	call.Action.To = nil
	call.Action.Input = nil
	call.Action.Gas = nil

	call.Result = nil
}

func (t *callParityTracer) convertErrorToParity(call *callParityFrame) {
	if call.Error == "" {
		return
	}

	if parityError, ok := parityErrorMapping[call.Error]; ok {
		call.Error = parityError
		call.Result = nil
	} else {
		for gethError, parityError := range parityErrorMappingStartingWith {
			if strings.HasPrefix(call.Error, gethError) {
				call.Error = parityError
				call.Result = nil
			}
		}
	}
}
