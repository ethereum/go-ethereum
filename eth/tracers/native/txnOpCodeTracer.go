package native

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/log"
)

func init() {
	register("txnOpCodeTracer", newtxnOpCodeTracer)
}

type callFrameBN struct {
	Type        string        `json:"type"`
	From        string        `json:"from"`
	To          string        `json:"to,omitempty"`
	Value       string        `json:"value,omitempty"`
	Gas         string        `json:"gas"`
	GasUsed     string        `json:"gasUsed"`
	Input       string        `json:"input"`
	Output      string        `json:"output,omitempty"`
	Error       string        `json:"error,omitempty"`
	ErrorReason string        `json:"errorReason,omitempty"`
	Calls       []callFrameBN `json:"calls,omitempty"`

	// Added fields from 'callFrame' in 'call.go'
	gasIn   uint64
	gasCost uint64
	Time    string `json:"time,omitempty"`

	// TODO: Investigate precompiles usage, could reduce latency as they are "those are just fancy opcodes" - 4byte.go
	// activePrecompiles []common.Address // Updated on CaptureStart based on given rules

}

// txnOpCodeTracer is a go implementation of the Tracer interface which
// only returns a restricted trace of a transaction consisting of transaction
// op codes and relevant gas data.
// This is intended for Blocknative usage.
type txnOpCodeTracer struct {
	env       *vm.EVM       // EVM context for execution of transaction to occur within
	callStack []callFrameBN // Data structure for op codes making up our trace
	interrupt uint32        // Atomic flag to signal execution interruption
	reason    error         // Textual reason for the interruption (not always specific for us)
}

// newtxnOpCodeTracer returns a new txnOpCodeTracer tracer.
func newtxnOpCodeTracer(ctx *tracers.Context) tracers.Tracer {
	// First callframe contains tx context info
	// and is populated on start and end.
	return &txnOpCodeTracer{callStack: make([]callFrameBN, 1)}
}

// GetResult returns an empty json object.
func (t *txnOpCodeTracer) GetResult() (json.RawMessage, error) {
	if len(t.callStack) != 1 {
		return nil, errors.New("incorrect number of top-level calls")
	}
	res, err := json.Marshal(t.callStack[0])
	if err != nil {
		return nil, err
	}
	return json.RawMessage(res), t.reason
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *txnOpCodeTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.env = env
	// This is the initial call
	t.callStack[0] = callFrameBN{
		Type:  "CALL",
		From:  addrToHex(from),
		To:    addrToHex(to),
		Input: bytesToHex(input),
		Gas:   uintToHex(gas),
		Value: bigToHex(value),
	}
	if create {
		// TODO: Here we can note creation of contracts for potential future tracing
		t.callStack[0].Type = "CREATE"
	}
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *txnOpCodeTracer) CaptureEnd(output []byte, gasUsed uint64, time time.Duration, err error) {
	// Collect final gasUsed
	t.callStack[0].GasUsed = uintToHex(gasUsed)

	// Add total time duration for this trace request
	t.callStack[0].Time = fmt.Sprintf("%v", time)

	// This is the final output of a call
	if err != nil {
		t.callStack[0].Error = err.Error()
		if err.Error() == "execution reverted" && len(output) > 0 {
			t.callStack[0].Output = bytesToHex(output)

			// This revert reason is found via the standard introduced in v0.8.4
			// It uses a ABI with the method Error(string)
			// This is the top level call, internal txns may fail while top level succeeds still
			revertReason, _ := abi.UnpackRevert(output)
			t.callStack[0].ErrorReason = revertReason
		}
	} else {
		// TODO: This output is for the originally called contract, we can use the ABI to decode this for useful information
		// ie: there are custom error types in ABIs since 0.8.4 which will turn up here
		t.callStack[0].Output = bytesToHex(output)
	}
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *txnOpCodeTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	defer func() {
		if r := recover(); r != nil {
			t.callStack[depth].Error = "internal failure"
			log.Warn("Panic during trace. Recovered.", "err", r)
		}
	}()

	// TODO: Here we can check for specific op codes that may interest us
	// Op codes we like at BN are:
	// CREATE, CREATE2
	// SELFDESTRUCT
	// CALL, CALLCODE, DELEGATECALL, STATICCALL (picked up by CaptureEnter)
	// REVERT
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (t *txnOpCodeTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
	// The err here is generated by geth, not by contract error logging
}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *txnOpCodeTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// Skip if tracing was interrupted
	if atomic.LoadUint32(&t.interrupt) > 0 {
		t.env.Cancel()
		return
	}

	// Apart from the starting call detected by CaptureStart, here we track every new transaction opcode
	call := callFrameBN{
		Type:  typ.String(),
		From:  addrToHex(from),
		To:    addrToHex(to),
		Input: bytesToHex(input),
		Gas:   uintToHex(gas),
		Value: bigToHex(value),
	}
	t.callStack = append(t.callStack, call)

	// Todo: Can add a decode request here from OWL in future
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *txnOpCodeTracer) CaptureExit(output []byte, gasUsed uint64, err error) {

	size := len(t.callStack)
	if size <= 1 {
		return
	}
	// pop call
	call := t.callStack[size-1]
	t.callStack = t.callStack[:size-1]
	size -= 1

	call.GasUsed = uintToHex(gasUsed)
	if err == nil {
		call.Output = bytesToHex(output)
	} else {
		call.Error = err.Error()
		if call.Type == "CREATE" || call.Type == "CREATE2" {
			call.To = ""
		}
	}
	t.callStack[size-1].Calls = append(t.callStack[size-1].Calls, call)
}

func (*txnOpCodeTracer) CaptureTxStart(gasLimit uint64) {
}

func (*txnOpCodeTracer) CaptureTxEnd(restGas uint64) {}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *txnOpCodeTracer) Stop(err error) {
	t.reason = err
	atomic.StoreUint32(&t.interrupt, 1)
}
