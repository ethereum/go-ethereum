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
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

// go:generate go run github.com/fjl/gencodec -type flatCallFrame -field-override flatCallFrameMarshaling -out gen_flatcallframe_json.go

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

// callParityFrame is the result of a callParityTracerParity run.
type flatCallFrame struct {
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

// type flatCallFrameMarshaling struct {
// 	Action              CallTraceParityAction `json:"action"`
// 	BlockHash           *common.Hash          `json:"-"`
// 	BlockNumber         uint64                `json:"-"`
// 	Error               string                `json:"error,omitempty"`
// 	Result              CallTraceParityResult `json:"result,omitempty"`
// 	Subtraces           int                   `json:"subtraces"`
// 	TraceAddress        []int                 `json:"traceAddress"`
// 	TransactionHash     *common.Hash          `json:"-"`
// 	TransactionPosition *uint64               `json:"-"`
// 	Type                string                `json:"type"`
// 	Time                string                `json:"-"`
// }

// flatCallTracer is a go implementation of the Tracer interface which
// runs multiple tracers in one go.
type flatCallTracer struct {
	tracer            tracers.Tracer
	env               *vm.EVM
	config            flatCallTracerConfig
	ctx               *tracers.Context // Holds tracer context data
	callstack         []callParityFrame
	interrupt         uint32           // Atomic flag to signal execution interruption
	reason            error            // Textual reason for the interruption
	activePrecompiles []common.Address // Updated on CaptureStart based on given rules
}

type flatCallTracerConfig struct {
	// OnlyTopCall bool `json:"onlyTopCall"` // If true, call tracer won't collect any subcalls
	// WithLog     bool `json:"withLog"`     // If true, call tracer will collect event logs
}

// newFlatCallTracer returns a new mux tracer.
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

	return &flatCallTracer{tracer: tracer, ctx: ctx, config: config}, nil
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *flatCallTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.tracer.CaptureStart(env, from, to, create, input, gas, value)
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *flatCallTracer) CaptureEnd(output []byte, gasUsed uint64, elapsed time.Duration, err error) {
	t.tracer.CaptureEnd(output, gasUsed, elapsed, err)
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
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *flatCallTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	t.tracer.CaptureExit(output, gasUsed, err)
}

func (t *flatCallTracer) CaptureTxStart(gasLimit uint64) {
	t.tracer.CaptureTxStart(gasLimit)
}

func (t *flatCallTracer) CaptureTxEnd(restGas uint64) {
	t.tracer.CaptureTxEnd(restGas)
}

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

	// not so nice way to read the TypeString from the json
	traceResultMarshaled := new(callFrameMarshaling)
	err = json.Unmarshal(traceResultJson, &traceResultMarshaled)
	if err != nil {
		return nil, err
	}

	traceResult.Type = vm.StringToOp(traceResultMarshaled.TypeString)

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
	// fmt.Println("input:", input)
	// finalize: function(call, extraCtx, traceAddress) {
	// 	var data;
	// 	if (call.type == "CREATE" || call.type == "CREATE2") {
	// 		data = this.createResult(call);

	// 		// update after callResult so as it affects only the root type
	// 		call.type = "CREATE";
	// 	} else if (call.type == "SELFDESTRUCT") {
	// 		call.type = "SUICIDE";
	// 		data = this.suicideResult(call);
	// 	} else {
	// 		data = this.callResult(call);

	// 		// update after callResult so as it affects only the root type
	// 		if (call.type == "CALLCODE" || call.type == "DELEGATECALL" || call.type == "STATICCALL") {
	// 			call.type = "CALL";
	// 		}
	// 	}

	// 	traceAddress = traceAddress || [];
	// 	var sorted = {
	// 		type: call.type.toLowerCase(),
	// 		action: data.action,
	// 		result: data.result,
	// 		error: call.error,
	// 		traceAddress: traceAddress,
	// 		subtraces: 0,
	// 		transactionPosition: extraCtx.transactionPosition,
	// 		transactionHash: extraCtx.transactionHash,
	// 		blockNumber: extraCtx.blockNumber,
	// 		blockHash: extraCtx.blockHash,
	// 		time: call.time,
	// 	}

	gasHex := hexutil.Uint64(input.Gas)
	gasUsedHex := hexutil.Uint64(input.GasUsed)
	valueHex := hexutil.Big{}
	if input.Value != nil {
		valueHex = hexutil.Big(*input.Value)
	}

	frame := flatCallFrame{
		Action: CallTraceParityAction{
			From:  &input.From,
			Gas:   &gasHex,
			Value: &valueHex,
		},
		Result: &CallTraceParityResult{
			GasUsed: &gasUsedHex,
		},
		// Action: input.Action,
		Error: input.Error,
		// Result: input.Result,
		// Subtraces:    input.Subtraces,
		TraceAddress: traceAddress,
		// Type: strings.ToLower(input.Type.String()),
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

	output = append(output, frame)
	// 	if (sorted.error !== undefined) {
	// 		if (this.parityErrorMapping.hasOwnProperty(sorted.error)) {
	// 			sorted.error = this.parityErrorMapping[sorted.error];
	// 			delete sorted.result;
	// 		} else {
	// 			for (var searchKey in this.parityErrorMappingStartingWith) {
	// 				if (this.parityErrorMappingStartingWith.hasOwnProperty(searchKey) && sorted.error.indexOf(searchKey) > -1) {
	// 					sorted.error = this.parityErrorMappingStartingWith[searchKey];
	// 					delete sorted.result;
	// 				}
	// 			}
	// 		}
	// 	}

	frame.Subtraces = len(input.Calls)

	if len(input.Calls) > 0 {
		for i, childCall := range input.Calls {
			traceAddress = append(traceAddress, i)
			flat, err := t.processOutput(&childCall, traceAddress)
			if err != nil {
				return nil, err
			}
			output = append(output, flat...)
		}
	}

	// 	var calls = call.calls;
	// 	if (calls !== undefined) {
	// 		sorted["subtraces"] = calls.length;
	// 	}

	// 	var results = [sorted];

	// 	if (calls !== undefined) {
	// 		for (var i=0; i<calls.length; i++) {
	// 			var childCall = calls[i];

	// 			// Delegatecall uses the value from parent
	// 			if ((childCall.type == "DELEGATECALL" || childCall.type == "STATICCALL") && typeof childCall.value === "undefined") {
	// 				childCall.value = call.value;
	// 			}

	// 			results = results.concat(this.finalize(childCall, extraCtx, traceAddress.concat([i])));
	// 		}
	// 	}
	// 	return results;
	// },

	// output = append(output, flatCallFrame{
	// 	// Action:              input.Action,
	// 	// BlockHash:           input.BlockHash,
	// 	// BlockNumber:         input.BlockNumber,
	// 	// Error:               input.Error,
	// 	// Result:              input.Result,
	// 	// Subtraces:           input.Subtraces,
	// 	// TraceAddress:        input.TraceAddress,
	// 	// TransactionHash:     input.TransactionHash,
	// 	// TransactionPosition: input.TransactionPosition,
	// 	Type: strings.ToLower(input.Type.String()),
	// })

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

func (t *flatCallTracer) convertErrorToParity(call *callParityFrame) {
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
