// Copyright 2021 XDC Network
// This file is part of the XDC library.

package native

import (
	"encoding/json"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

func init() {
	tracers.DefaultDirectory.Register("xdcTracer", newXDCTracer, false)
}

// xdcTracer is a native tracer for XDC-specific operations
type xdcTracer struct {
	env       *vm.EVM
	config    xdcTracerConfig
	gasLimit  uint64
	interrupt atomic.Bool
	reason    error

	// Results
	result    *XDCTraceResult
	callStack []*xdcCall
}

// xdcTracerConfig holds configuration for the XDC tracer
type xdcTracerConfig struct {
	TraceInternalCalls bool `json:"traceInternalCalls"`
	TraceStorage       bool `json:"traceStorage"`
	TraceRewards       bool `json:"traceRewards"`
}

// XDCTraceResult holds the trace result
type XDCTraceResult struct {
	Type         string         `json:"type"`
	From         common.Address `json:"from"`
	To           common.Address `json:"to"`
	Value        *big.Int       `json:"value"`
	Gas          uint64         `json:"gas"`
	GasUsed      uint64         `json:"gasUsed"`
	Input        []byte         `json:"input"`
	Output       []byte         `json:"output"`
	Error        string         `json:"error,omitempty"`
	RevertReason string         `json:"revertReason,omitempty"`
	Calls        []*xdcCall     `json:"calls,omitempty"`
}

// xdcCall represents an internal call
type xdcCall struct {
	Type    string         `json:"type"`
	From    common.Address `json:"from"`
	To      common.Address `json:"to"`
	Value   *big.Int       `json:"value,omitempty"`
	Gas     uint64         `json:"gas"`
	GasUsed uint64         `json:"gasUsed"`
	Input   []byte         `json:"input,omitempty"`
	Output  []byte         `json:"output,omitempty"`
	Error   string         `json:"error,omitempty"`
	Calls   []*xdcCall     `json:"calls,omitempty"`
}

// newXDCTracer creates a new XDC tracer
func newXDCTracer(ctx *tracers.Context, cfg json.RawMessage) (tracers.Tracer, error) {
	var config xdcTracerConfig
	if cfg != nil {
		if err := json.Unmarshal(cfg, &config); err != nil {
			return nil, err
		}
	}
	return &xdcTracer{
		config: config,
		result: &XDCTraceResult{},
	}, nil
}

// CaptureStart implements the EVMLogger interface
func (t *xdcTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.env = env
	t.gasLimit = gas

	t.result.From = from
	t.result.To = to
	t.result.Input = input
	t.result.Gas = gas
	if value != nil {
		t.result.Value = new(big.Int).Set(value)
	}

	if create {
		t.result.Type = "CREATE"
	} else {
		t.result.Type = "CALL"
	}
}

// CaptureEnd implements the EVMLogger interface
func (t *xdcTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	t.result.Output = output
	t.result.GasUsed = gasUsed
	if err != nil {
		t.result.Error = err.Error()
	}
}

// CaptureState implements the EVMLogger interface
func (t *xdcTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	// Not tracking individual opcodes in this tracer
}

// CaptureFault implements the EVMLogger interface
func (t *xdcTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	// Capture fault information
}

// CaptureEnter implements the EVMLogger interface
func (t *xdcTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	if !t.config.TraceInternalCalls {
		return
	}

	call := &xdcCall{
		Type:  typ.String(),
		From:  from,
		To:    to,
		Gas:   gas,
		Input: input,
	}
	if value != nil {
		call.Value = new(big.Int).Set(value)
	}

	t.callStack = append(t.callStack, call)
}

// CaptureExit implements the EVMLogger interface
func (t *xdcTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	if !t.config.TraceInternalCalls || len(t.callStack) == 0 {
		return
	}

	// Pop the last call from the stack
	size := len(t.callStack)
	call := t.callStack[size-1]
	t.callStack = t.callStack[:size-1]

	call.Output = output
	call.GasUsed = gasUsed
	if err != nil {
		call.Error = err.Error()
	}

	// Add to parent or result
	if len(t.callStack) > 0 {
		parent := t.callStack[len(t.callStack)-1]
		parent.Calls = append(parent.Calls, call)
	} else {
		t.result.Calls = append(t.result.Calls, call)
	}
}

// CaptureTxStart implements the EVMLogger interface
func (t *xdcTracer) CaptureTxStart(gasLimit uint64) {
	t.gasLimit = gasLimit
}

// CaptureTxEnd implements the EVMLogger interface
func (t *xdcTracer) CaptureTxEnd(restGas uint64) {
	t.result.GasUsed = t.gasLimit - restGas
}

// GetResult returns the trace result
func (t *xdcTracer) GetResult() (json.RawMessage, error) {
	return json.Marshal(t.result)
}

// Stop terminates the tracer
func (t *xdcTracer) Stop(err error) {
	t.reason = err
	t.interrupt.Store(true)
}

// XDCRewardTracer traces validator rewards
type XDCRewardTracer struct {
	rewards map[common.Address]*big.Int
}

// NewXDCRewardTracer creates a new reward tracer
func NewXDCRewardTracer() *XDCRewardTracer {
	return &XDCRewardTracer{
		rewards: make(map[common.Address]*big.Int),
	}
}

// AddReward adds a reward for a validator
func (t *XDCRewardTracer) AddReward(validator common.Address, amount *big.Int) {
	if existing, ok := t.rewards[validator]; ok {
		t.rewards[validator] = new(big.Int).Add(existing, amount)
	} else {
		t.rewards[validator] = new(big.Int).Set(amount)
	}
}

// GetRewards returns all rewards
func (t *XDCRewardTracer) GetRewards() map[common.Address]*big.Int {
	result := make(map[common.Address]*big.Int)
	for addr, amount := range t.rewards {
		result[addr] = new(big.Int).Set(amount)
	}
	return result
}

// XDCPenaltyTracer traces validator penalties
type XDCPenaltyTracer struct {
	penalties []XDCPenalty
}

// XDCPenalty represents a penalty
type XDCPenalty struct {
	Validator common.Address
	Amount    *big.Int
	Reason    string
	Block     uint64
}

// NewXDCPenaltyTracer creates a new penalty tracer
func NewXDCPenaltyTracer() *XDCPenaltyTracer {
	return &XDCPenaltyTracer{
		penalties: make([]XDCPenalty, 0),
	}
}

// AddPenalty adds a penalty
func (t *XDCPenaltyTracer) AddPenalty(validator common.Address, amount *big.Int, reason string, block uint64) {
	t.penalties = append(t.penalties, XDCPenalty{
		Validator: validator,
		Amount:    new(big.Int).Set(amount),
		Reason:    reason,
		Block:     block,
	})
}

// GetPenalties returns all penalties
func (t *XDCPenaltyTracer) GetPenalties() []XDCPenalty {
	return t.penalties
}
