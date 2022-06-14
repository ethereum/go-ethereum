package logger

import (
	"encoding/json"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

type ACLDiffTracer struct {
	env       *vm.EVM
	tracer    *AccessListTracer
	txACL     types.AccessList
	interrupt uint32 // Atomic flag to signal execution interruption
	reason    error  // Textual reason for the interruption
}

func NewACLDiffTracer() *ACLDiffTracer {
	return &ACLDiffTracer{}
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *ACLDiffTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.env = env
	rules := env.ChainConfig().Rules(env.Context.BlockNumber, env.Context.Random != nil)
	precompiles := vm.ActivePrecompiles(rules)
	t.tracer = NewAccessListTracer(nil, from, to, precompiles)
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *ACLDiffTracer) CaptureEnd(output []byte, gasUsed uint64, _ time.Duration, err error) {
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *ACLDiffTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	// Skip if tracing was interrupted
	if atomic.LoadUint32(&t.interrupt) > 0 {
		t.env.Cancel()
		return
	}
	t.tracer.CaptureState(pc, op, gas, cost, scope, rData, depth, err)
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (t *ACLDiffTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *ACLDiffTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// Skip if tracing was interrupted
	if atomic.LoadUint32(&t.interrupt) > 0 {
		t.env.Cancel()
		return
	}

}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *ACLDiffTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
}

func (t *ACLDiffTracer) CaptureTxStart(gasLimit uint64, acl types.AccessList) {
	t.txACL = acl
}

func (t *ACLDiffTracer) CaptureTxEnd(restGas uint64) {}

// GetResult returns the json-encoded nested list of call traces, and any
// error arising from the encoding or forceful termination (via `Stop`).
func (t *ACLDiffTracer) GetResult() (json.RawMessage, error) {
	if len(t.txACL) == 0 {
		return nil, fmt.Errorf("Transaction not of type 1")
	}
	touched := t.tracer.list
	txACL := newAccessListFrom(t.txACL, nil)
	excess := txACL.diff(touched).accessList()
	unannounced := touched.diff(txACL).accessList()
	res, err := json.Marshal(map[string]interface{}{"excess": excess, "unannounced": unannounced})
	if err != nil {
		return nil, err
	}
	return json.RawMessage(res), t.reason

}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *ACLDiffTracer) Stop(err error) {
	t.reason = err
	atomic.StoreUint32(&t.interrupt, 1)
}
