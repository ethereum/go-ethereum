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
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

func init() {
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
	Action              flatCallTraceAction  `json:"action"`
	BlockHash           *common.Hash         `json:"blockHash"`
	BlockNumber         uint64               `json:"blockNumber"`
	Error               string               `json:"error,omitempty"`
	Result              *flatCallTraceResult `json:"result,omitempty"`
	Subtraces           int                  `json:"subtraces"`
	TraceAddress        []int                `json:"traceAddress"`
	TransactionHash     *common.Hash         `json:"transactionHash"`
	TransactionPosition *uint64              `json:"transactionPosition"`
	Type                string               `json:"type"`
}

type flatCallTraceAction struct {
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

type flatCallTraceResult struct {
	Address *common.Address `json:"address,omitempty"`
	Code    *hexutil.Bytes  `json:"code,omitempty"`
	GasUsed *hexutil.Uint64 `json:"gasUsed,omitempty"`
	Output  *hexutil.Bytes  `json:"output,omitempty"`
}

// flatCallTracer reports call frame information of a tx in a flat format, i.e.
// as opposed to the nested format of `callTracer`.
type flatCallTracer struct {
	tracer            *callTracer
	config            flatCallTracerConfig
	ctx               *tracers.Context // Holds tracer context data
	reason            error            // Textual reason for the interruption
	activePrecompiles []common.Address // Updated on CaptureStart based on given rules
}

type flatCallTracerConfig struct {
	ConvertedParityErrors bool `json:"convertedParityErrors"` // If true, call tracer converts errors to parity format
}

// newFlatCallTracer returns a new flatCallTracer.
func newFlatCallTracer(ctx *tracers.Context, cfg json.RawMessage) (tracers.Tracer, error) {
	var config flatCallTracerConfig
	if cfg != nil {
		if err := json.Unmarshal(cfg, &config); err != nil {
			return nil, err
		}
	}

	tracer, err := tracers.New("callTracer", ctx, cfg)
	if err != nil {
		return nil, err
	}
	t, ok := tracer.(*callTracer)
	if !ok {
		return nil, errors.New("internal error: embedded tracer has wrong type")
	}

	return &flatCallTracer{tracer: t, ctx: ctx, config: config}, nil
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *flatCallTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.tracer.CaptureStart(env, from, to, create, input, gas, value)
	// Update list of precompiles based on current block
	rules := env.ChainConfig().Rules(env.Context.BlockNumber, env.Context.Random != nil)
	t.activePrecompiles = vm.ActivePrecompiles(rules)
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *flatCallTracer) CaptureEnd(output []byte, gasUsed uint64, elapsed time.Duration, err error) {
	t.tracer.CaptureEnd(output, gasUsed, elapsed, err)
	// Parity trace considers only reports the gas used during the top call frame which doesn't include
	// tx processing such as intrinsic gas and refunds.
	t.tracer.callstack[0].GasUsed = gasUsed
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *flatCallTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	t.tracer.CaptureState(pc, op, gas, cost, scope, rData, depth, err)
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (t *flatCallTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	t.tracer.CaptureFault(pc, op, gas, cost, scope, depth, err)
}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *flatCallTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	t.tracer.CaptureEnter(typ, from, to, input, gas, value)
	// Delegatecall has same value as parent call.
	// CallTracer doesn't report this "inherited" value.
	if typ == vm.DELEGATECALL {
		size := len(t.tracer.callstack)
		if size < 2 {
			return
		}
		t.tracer.callstack[size-1].Value = t.tracer.callstack[size-2].Value
	}
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *flatCallTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	t.tracer.CaptureExit(output, gasUsed, err)
	// Parity traces don't include CALL/STATICCALLs to precompiles.
	var (
		// call has been nested in parent
		parent = t.tracer.callstack[len(t.tracer.callstack)-1]
		call   = parent.Calls[len(parent.Calls)-1]
		typ    = call.Type
		to     = call.To
	)
	if typ == vm.CALL || typ == vm.STATICCALL {
		if t.isPrecompiled(to) {
			t.tracer.callstack[len(t.tracer.callstack)-1].Calls = parent.Calls[:len(parent.Calls)-1]
		}
	}
}

func (t *flatCallTracer) CaptureTxStart(gasLimit uint64) {}

func (t *flatCallTracer) CaptureTxEnd(restGas uint64) {}

// GetResult returns an empty json object.
func (t *flatCallTracer) GetResult() (json.RawMessage, error) {
	traceResultJson, err := t.tracer.GetResult()
	if err != nil {
		return nil, err
	}

	traceResult := new(callFrame)
	err = json.Unmarshal(traceResultJson, &traceResult)
	if err != nil {
		return nil, err
	}

	flat, err := t.processOutput(traceResult, []int{})
	if err != nil {
		return nil, err
	}

	res, err := json.Marshal(flat)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(res), t.reason
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *flatCallTracer) Stop(err error) {
	t.tracer.Stop(err)
}

func (t *flatCallTracer) processOutput(input *callFrame, traceAddress []int) (output []flatCallFrame, err error) {
	gasHex := hexutil.Uint64(input.Gas)
	gasUsedHex := hexutil.Uint64(input.GasUsed)
	valueHex := hexutil.Big{}
	if input.Value != nil {
		valueHex = hexutil.Big(*input.Value)
	}

	// copy addresses
	to := input.To
	from := input.From
	frame := flatCallFrame{
		Type: strings.ToLower(input.Type.String()),
		Action: flatCallTraceAction{
			From:  &from,
			To:    &to,
			Gas:   &gasHex,
			Value: &valueHex,
		},
		Result: &flatCallTraceResult{
			GasUsed: &gasUsedHex,
		},
		Error:        input.Error,
		TraceAddress: traceAddress,
	}

	// typ := vm.StringToOp(strings.ToUpper(call.Type))
	if input.Type == vm.CREATE || input.Type == vm.CREATE2 {
		t.formatCreateResult(&frame, input)
	} else if input.Type == vm.SELFDESTRUCT {
		t.formatSuicideResult(&frame, input)
	} else {
		t.formatCallResult(&frame, input)
	}

	t.fillCallFrameFromContext(&frame)

	if t.config.ConvertedParityErrors {
		t.convertErrorToParity(&frame)
	}

	frame.Subtraces = len(input.Calls)

	output = append(output, frame)

	if len(input.Calls) > 0 {
		for i, childCall := range input.Calls {
			var childTraceAddress []int
			childTraceAddress = append(childTraceAddress, traceAddress...)
			childTraceAddress = append(childTraceAddress, i)
			flat, err := t.processOutput(&childCall, childTraceAddress)
			if err != nil {
				return nil, err
			}
			output = append(output, flat...)
		}
	}

	return output, nil
}

func (t *flatCallTracer) fillCallFrameFromContext(callFrame *flatCallFrame) {
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

func (t *flatCallTracer) formatCreateResult(call *flatCallFrame, input *callFrame) {
	call.Type = strings.ToLower(vm.CREATE.String())

	init := hexutil.Bytes(input.Input[:])
	call.Action.Init = &init

	call.Result.Address = &input.To

	code := hexutil.Bytes(input.Output[:])
	call.Result.Code = &code

	call.Action.To = nil
	call.Action.Input = nil

	call.Result.Output = nil
}

func (t *flatCallTracer) formatCallResult(call *flatCallFrame, input *callFrame) {
	call.Action.CallType = strings.ToLower(input.Type.String())

	// update after callResult so as it affects only the root type
	if input.Type == vm.CALLCODE || input.Type == vm.DELEGATECALL || input.Type == vm.STATICCALL {
		call.Type = strings.ToLower(vm.CALL.String())
	}

	actionInput := hexutil.Bytes(input.Input[:])
	call.Action.Input = &actionInput

	resultOutput := hexutil.Bytes(input.Output[:])
	call.Result.Output = &resultOutput
}

func (t *flatCallTracer) formatSuicideResult(call *flatCallFrame, input *callFrame) {
	// this is using the old opcode, in order we maintain parity compatibility
	call.Type = "suicide"

	call.Action.SelfDestructed = &input.From

	balanceHex := hexutil.Big(*input.Value)
	call.Action.Balance = &balanceHex

	call.Action.Value = nil

	call.Action.RefundAddress = &input.To

	call.Action.From = nil
	call.Action.To = nil
	call.Action.Input = nil
	call.Action.Gas = nil

	call.Result = nil
}

func (t *flatCallTracer) convertErrorToParity(call *flatCallFrame) {
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

// isPrecompiled returns whether the addr is a precompile.
func (t *flatCallTracer) isPrecompiled(addr common.Address) bool {
	for _, p := range t.activePrecompiles {
		if p == addr {
			return true
		}
	}
	return false
}
